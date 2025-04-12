package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"solfa-back/lib"
	"solfa-back/models"
	"fmt"
	"time"
	"github.com/minio/minio-go/v7"
	"encoding/json"
	"strings"
	"path/filepath"
)

func UploadPartitionHandler(c *gin.Context) {
	// Vérifier le token
	claims, err := lib.ExtractUserClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
		return
	}

	// Récupérer les informations JSON et le fichier
	var request struct {
		Title       string `json:"title" binding:"required"`
		Composer    string `json:"composer"`
		Genre       string `json:"genre"`
		Category    string `json:"category"`
		ReleaseDate string `json:"release_date"`
	}

	// Lier la requête JSON
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error binding request": err.Error()})
		return
	}

	// Vérifier si la partition existe déjà dans Elasticsearch
	partitionExists, existingPartition := lib.SearchPartitionByFields(models.Partition{
		Title:    request.Title,
		Composer: request.Composer,
		Genre:    request.Genre,
		Category: request.Category,
	})
	
	if partitionExists {
		// Si la partition existe déjà, retourner ses informations
		c.JSON(http.StatusConflict, gin.H{
			"message":           "Partition déjà existante.",
			"existing_partition": existingPartition,
		})
		return
	}

	// Récupérer le fichier envoyé
	file, err := c.FormFile("partition_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Erreur lors de l'upload du fichier"})
		return
	}

	// Ouvrir le fichier temporaire
	srcFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'ouvrir le fichier"})
		return
	}
	defer srcFile.Close()

	// Créer un nom unique pour le fichier dans Minio
	filePath := fmt.Sprintf("partitions/%s_%s", time.Now().Format("20060102150405"), file.Filename)

	// Télécharger le fichier sur Minio
	_, err = lib.MinioClient.PutObject(c, "solfa", filePath, srcFile, file.Size, minio.PutObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur de téléchargement sur Minio " + err.Error()})
		return
	}

	var parsedDate time.Time
	if request.ReleaseDate != "" {
		parsedDate, err = time.Parse("2006-01-02", request.ReleaseDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format de date invalide, utilisez YYYY-MM-DD"})
			return
		}
	}

	// Enregistrer les informations dans PostgreSQL avec l'état 'staging'
	partition := models.Partition{
		Title:       request.Title,
		Composer:    request.Composer,
		Genre:       request.Genre,
		Category:    request.Category,
		ReleaseDate: parsedDate,
		Path:        filePath, // Le chemin du fichier dans Minio
		Status:      "validated", // Par défaut, la partition est en état de staging
		ValidatedBy: "", // L'email de l'utilisateur qui valide la partition
	}

	// Insérer la partition dans PostgreSQL
	if err := lib.DB.Create(&partition).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement dans la base de données"})
		return
	}

	// Indexer la partition dans Elasticsearch
	lib.IndexPartitionInES(partition)
	lib.LogAction("upload", claims.Email + " | " + partition.Title)

	// Réponse de succès
	c.JSON(http.StatusOK, gin.H{
		"message": "Partition uploadée avec succès, en attente de validation.",
		"file":    filePath,
	})
}



func SearchPartitionsHandler(c *gin.Context) {
    query := c.Query("q") // Récupère la requête de l'utilisateur

    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'q' est requis"})
        return
    }

    // Construction de la requête Elasticsearch
    searchQuery := fmt.Sprintf(`{
        "query": {
            "multi_match": {
                "query": "%s",
                "fields": ["title", "composer", "genre", "category"],
                "type": "best_fields",
                "fuzziness": "AUTO"
            }
        }
    }`, query)

    // Exécution de la requête
    res, err := lib.ESClient.Search(
        lib.ESClient.Search.WithIndex("partitions"),
        lib.ESClient.Search.WithBody(strings.NewReader(searchQuery)),
    )

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur Elasticsearch"})
        return
    }
    defer res.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur de décodage Elasticsearch"})
        return
    }

	lib.LogAction("search", query)
    hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
    c.JSON(http.StatusOK, gin.H{"results": hits})
}


func ValidatePartitionHandler(c *gin.Context) {
	// Récupérer l'email de l'utilisateur connecté
	claims, err := lib.ExtractUserClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
		return
	}

	// Vérifier si l'utilisateur est un administrateur
	if !claims.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à effectuer cette action"})
		return
	}

	// Récupérer l'ID de la partition
	partitionID := c.Param("id")

	// Récupérer la partition
	var partition models.Partition
	if err := lib.DB.First(&partition, partitionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Partition non trouvée"})
		return
	}

	// Mettre à jour l'état de la partition
	partition.Status = "validated"
	partition.ValidatedBy = claims.Email

	if err := lib.DB.Save(&partition).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la validation"})
		return
	}

	lib.UpdatePartitionStatus(partition, "validated")

	lib.LogAction("validate_partition", claims.Email)

	// Réponse de succès
	c.JSON(http.StatusOK, gin.H{"message": "Partition validée avec succès"})
}


// DownloadPartitionHandler gère le téléchargement d'une partition depuis MinIO
func DownloadPartitionHandler(c *gin.Context) {
    // Récupérer le paramètre "path" de la requête
    path := c.Query("path")
    if path == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Le paramètre 'path' est requis"})
        return
    }

    // Nom du bucket MinIO (conformément à UploadPartitionHandler)
    bucketName := "solfa"

    // Vérifier si l'objet existe dans le bucket MinIO
    _, err := lib.MinioClient.StatObject(c, bucketName, path, minio.StatObjectOptions{})
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Fichier non trouvé dans MinIO"})
        return
    }

    // Récupérer l'objet depuis MinIO
    object, err := lib.MinioClient.GetObject(c, bucketName, path, minio.GetObjectOptions{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération du fichier depuis MinIO"})
        return
    }
    defer object.Close()

    // Définir les en-têtes pour le téléchargement
    filename := filepath.Base(path)
    c.Header("Content-Disposition", "attachment; filename="+filename)
    c.Header("Content-Type", "application/pdf")

	lib.LogAction("download", path)
    // Servir le fichier
    c.DataFromReader(http.StatusOK, -1, "application/pdf", object, nil)
}


func DeletePartitionHandler(c *gin.Context) {
	// Récupérer l'ID de la partition
	partitionID := c.Param("id")

	claims , err := lib.ExtractUserClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
		return
	}

	// Vérifier si l'utilisateur est un administrateur
	if !claims.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à effectuer cette action"})
		return
	}

	// Vérifier si l'ID de la partition est valide
	if partitionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de partition manquant"})
		return
	}

	// Vérifier si la partition existe
	var partition models.Partition
	if err := lib.DB.First(&partition, partitionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Partition non trouvée"})
		return
	}

	// Supprimer la partition de PostgreSQL
	if err := lib.DB.Delete(&partition).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression de la partition"})
		return
	}

	// Supprimer la partition de MinIO
	err = lib.MinioClient.RemoveObject(c, "solfa", partition.Path, minio.RemoveObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression du fichier dans MinIO"})
		return
	}

	lib.DeletePartitionFromES(partitionID)

	lib.LogAction("delete_partition", claims.Email)

	c.JSON(http.StatusOK, gin.H{"message": "Partition supprimée avec succès"})
}


func GetAllPartitionsHandler(c *gin.Context) {
	// Recuperer toute la liste des partitions depuis Elasticsearch
	partitions, err := lib.GetAllPartitionsFromES()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des partitions"})
		return
	}

	// Récupérer l'email de l'utilisateur connecté si disponible dans le cas contraire on mettra une valeur par défaut
	claims, err := lib.ExtractUserClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
		return
	}
	// Vérifier si l'utilisateur est un administrateur
	if !claims.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à effectuer cette action"})
		return
	}
	// Log de l'action
	lib.LogAction("get_all_partitions", claims.Email)
	// Répondre avec la liste des partitions
	c.JSON(http.StatusOK, gin.H{"partitions": partitions})
}



func CheckTokenHandler(c *gin.Context) {
	// Vérifier le token
	claims, err := lib.ExtractUserClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
		return
	}

	lib.LogAction("checking_token_validity", claims.Email)
	// Répondre avec les informations de l'utilisateur
	c.JSON(http.StatusOK, gin.H{"user": claims})
}
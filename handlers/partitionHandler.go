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
	
)

func UploadPartitionHandler(c *gin.Context) {
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
		Status:      "staging", // Par défaut, la partition est en état de staging
		ValidatedBy: "", // L'email de l'utilisateur qui valide la partition
	}

	// Insérer la partition dans PostgreSQL
	if err := lib.DB.Create(&partition).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'enregistrement dans la base de données"})
		return
	}

	// Indexer la partition dans Elasticsearch
	lib.IndexPartitionInES(partition)

	// Réponse de succès
	c.JSON(http.StatusOK, gin.H{
		"message": "Partition uploadée avec succès, en attente de validation.",
		"file":    filePath,
	})
}


func ValidatePartitionHandler(c *gin.Context) {
    // Extraire le token de l'utilisateur
    userToken := c.GetHeader("Authorization")
    if userToken == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Token manquant"})
        return
    }

    // Extraire l'email à partir du token
	claims, err := lib.ExtractUserClaims(c)
    userEmail := claims.Email
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide " + err.Error()})
        return
    }

    // Récupérer l'ID de la partition depuis le corps de la requête
    var request struct {
        PartitionID string `json:"partition_id" binding:"required"`
    }
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
        return
    }

    // Rechercher la partition dans la base de données
    var partition models.Partition
    if err := lib.DB.First(&partition, "id = ?", request.PartitionID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Partition non trouvée"})
        return
    }
    
    // Mettre à jour le statut de la partition en 'validated'
    partition.Status = "validated"
    partition.ValidatedBy = userEmail // Enregistrer l'email de l'utilisateur validant la partition
    if err := lib.DB.Save(&partition).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour de la partition"})
        return
    }
    
    // Mettre à jour le statut dans Elasticsearch
    if err := lib.UpdatePartitionStatusInES(partition.ID, "validated"); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour dans Elasticsearch"})
        return
    }
    
	lib.LogAction("validate_partition", userEmail)

    // Réponse de succès
    c.JSON(http.StatusOK, gin.H{
        "message": "Partition validée avec succès",
        "partition": partition,
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

    hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
    c.JSON(http.StatusOK, gin.H{"results": hits})
}

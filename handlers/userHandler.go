package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"solfa-back/lib"
	"solfa-back/models"
)

const userNotFoundError = "Utilisateur non trouvé"

func GetSolfa(c *gin.Context) {
	c.JSON(200, gin.H{"message": "solfa"})
}



// GetCurrentUser récupère les infos du profil utilisateur connecté
func GetCurrentUser(c *gin.Context) {
	// Récupérer l'utilisateur à partir du token JWT
	claims, _ := lib.ExtractUserClaims(c)
	var user models.User

	// Chercher l'utilisateur par email
	if err := lib.DB.Where("email = ?", claims.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": userNotFoundError})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// UpdateCurrentUser met à jour le profil de l'utilisateur connecté
func UpdateCurrentUser(c *gin.Context) {
	claims, _ := lib.ExtractUserClaims(c)
	var user models.User

	// Vérifier si l'utilisateur existe
	if err := lib.DB.Where("email = ?", claims.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": userNotFoundError})
		return
	}

	// Lire les nouvelles valeurs
	var updateData map[string]string
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON invalide"})
		return
	}

	// Mettre à jour les champs (ex: username)
	if username, ok := updateData["username"]; ok {
		user.Username = username
	}

	// Sauvegarder les modifications
	if err := lib.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	lib.LogAction("update_profile", user.Email)

	c.JSON(http.StatusOK, gin.H{"message": "Profil mis à jour avec succès"})
}

// GetUserByID récupère les infos d'un utilisateur via son ID
func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	// Rechercher l'utilisateur par ID
	if err := lib.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": userNotFoundError})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"solfa-back/lib"
	"solfa-back/models"
	"crypto/rand"
	"encoding/hex"
	"net/smtp"
	"fmt"
	"strings"
)

// SignupRequest représente les données attendues dans la requête
type SignupRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func generateVerificationToken() (string, error) {
	bytes := make([]byte, 16) // Génère un token de 16 octets
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func sendVerificationEmail(toEmail string, token string) {
	from := "tonemail@example.com"
	password := "tonmotdepasse"
	smtpHost := "smtp.example.com"
	smtpPort := "587"

	link := fmt.Sprintf("http://localhost:8080/api/auth/verify?token=%s", token)

	body := fmt.Sprintf("Cliquez sur le lien pour vérifier votre compte : %s", link)

	auth := smtp.PlainAuth("", from, password, smtpHost)
	msg := []byte("Subject: Vérification de votre compte\n\n" + body)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{toEmail}, msg)
	if err != nil {
		fmt.Println("Erreur d'envoi d'email:", err)
	}
}

func SignupHandler(c *gin.Context) {
	var req SignupRequest


	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingUser models.User
	if err := lib.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "L'email est déjà utilisé"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur de hachage"})
		return
	}

	verificationToken, err := generateVerificationToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la génération du token"})
		return
	}

	user := models.User{
		Username:          req.Username,
		Email:             req.Email,
		Password:          string(hashedPassword),
		IsVerified:        true,
		VerificationToken: verificationToken,
	}

	lib.LogAction("signup", user.Email)

	if err := lib.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'inscription"})
		return
	}

	sendVerificationEmail(user.Email, verificationToken)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Utilisateur inscrit. Veuillez vérifier votre email.",
	})
}

func VerifyEmailHandler(c *gin.Context) {
	token := c.Query("token")

	var user models.User
	if err := lib.DB.Where("verification_token = ?", token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token invalide"})
		return
	}

	user.IsVerified = true
	user.VerificationToken = ""
	lib.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Email vérifié avec succès"})
}

////////////////////////////
// Structure pour la requête de login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Fonction pour gérer la connexion
func LoginHandler(c *gin.Context) {
	var req LoginRequest

	// Validation de la requête JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Vérifier si l'utilisateur existe
	var user models.User
	if err := lib.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}

	lib.LogAction("login", user.Email)

	// Vérifier si l'utilisateur a confirmé son email
	if !user.IsVerified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "L'email n'a pas été vérifié"})
		return
	}

	// Vérifier le mot de passe
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email ou mot de passe incorrect"})
		return
	}


	// Générer le token JWT
	token, err := lib.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la génération du token"})
		return
	}

	// Retourner les infos utilisateur + token
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"token":    token,
	})
}

////////////////////////////
// LogoutHandler gère la déconnexion des utilisateurs
func LogoutHandler(c *gin.Context) {
	// Récupérer le token JWT de l'en-tête Authorization
	tokenString := c.GetHeader("Authorization")

	// Vérifier si le token est présent
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token manquant"})
		return
	}

	// Supprimer le préfixe "Bearer " si présent dans le token
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parser et valider le token
	claims, err := lib.ParseJWT(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
		return
	}

	// Récupérer l'ID ou l'email de l'utilisateur à partir du token
	email := claims.Email

	// Si tu veux loguer l'utilisateur qui se déconnecte, tu peux ici :
	lib.LogAction("logout", email)

	// Retourner un message de succès
	c.JSON(http.StatusOK, gin.H{
		"message": "Déconnexion réussie",
		"email":   email,
	})
}


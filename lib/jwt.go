package lib

import (
	"time"
	"github.com/golang-jwt/jwt/v4"
	"solfa-back/models"
	"errors"
)

// Clé secrète pour signer le JWT
var jwtKey = []byte("tonsecretkey")  // Remplace par une clé secrète plus robuste en prod !

// Structure pour le token JWT
type Claims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// Fonction pour générer un JWT
func GenerateJWT(user models.User) (string, error) {
	// Définir les revendications du JWT
	claims := &Claims{
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "solfa-back",           // Issuer, peut être modifié
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Expiration après 24 heures
		},
	}

	// Créer le token JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Signer le token avec la clé secrète
	signedToken, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}


var JwtSecretKey = []byte("your-secret-key") // clé secrète utilisée pour signer les tokens


// Fonction pour parser et valider le token JWT
func ParseJWT(tokenString string) (*Claims, error) {
	// Parser le token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Vérifier que l'algorithme utilisé pour signer le token est bien celui attendu
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("méthode de signature invalide")
		}
		return JwtSecretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	// Récupérer les claims et les retourner
	if claims, ok := token.Claims.(*Claims); ok {
		return claims, nil
	}
		return nil, errors.New("token invalide")
	}
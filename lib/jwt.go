package lib

import (
	"context"
	"errors"
	"fmt"
	"solfa-back/models"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
)

// Clé secrète pour signer le JWT
var jwtKey = []byte("tonsecretkey") // Remplace par une clé secrète plus robuste en prod !

// Client Redis (optionnel, pour cache et liste noire)
var redisClient = redis.NewClient(&redis.Options{
    Addr: "http://147.79.114.72:30079",
})
var ctx = context.Background()

// Structure pour le token JWT
type Claims struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    IsAdmin  bool   `json:"is_admin"`
    JTI      string `json:"jti"`
    jwt.RegisteredClaims
}

// Générer un JWT
func GenerateJWT(user models.User) (string, error) {
    claims := &Claims{
        Username: user.Username,
        Email:    user.Email,
        JTI:      fmt.Sprintf("%d", time.Now().UnixNano()),
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    "solfa-back",
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
        IsAdmin: user.IsAdmin,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signedToken, err := token.SignedString(jwtKey)
    if err != nil {
        return "", err
    }
    return signedToken, nil
}

// Vérifier si un token est dans la liste noire
func isTokenBlacklisted(jti string) bool {
    _, err := redisClient.Get(ctx, "blacklist:"+jti).Result()
    return err == nil
}

// Parser et valider le token JWT avec vérification de l'utilisateur
func ParseJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("méthode de signature invalide")
        }
        return jwtKey, nil
    })

    if err != nil || !token.Valid {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok {
        return nil, errors.New("token invalide")
    }

    // Vérifier la liste noire
    if isTokenBlacklisted(claims.JTI) {
        return nil, errors.New("token révoqué")
    }

    // Vérifier si l'utilisateur existe encore dans la base de données
    user, err := GetUserByUsername(claims.Username)
    if err != nil || user == nil {
        // Si l'utilisateur n'existe pas, blacklister le token pour éviter d'autres vérifications inutiles
        ttl := time.Until(claims.ExpiresAt.Time)
        if ttl > 0 {
            redisClient.Set(ctx, "blacklist:"+claims.JTI, "revoked", ttl)
        }
        return nil, errors.New("utilisateur supprimé ou inexistant")
    }

    return claims, nil
}

// Ajouter un token à la liste noire
func BlacklistToken(tokenString string) error {
    claims, err := ParseJWT(tokenString)
    if err != nil {
        return err
    }

    ttl := time.Until(claims.ExpiresAt.Time)
    if ttl < 0 {
        ttl = 0
    }

    err = redisClient.Set(ctx, "blacklist:"+claims.JTI, "revoked", ttl).Err()
    if err != nil {
        return fmt.Errorf("erreur lors de l'ajout à la liste noire: %v", err)
    }
    return nil
}

// ExtractUserClaims
func ExtractUserClaims(c *gin.Context) (*Claims, error) {
    tokenString, err := c.Cookie("jwt_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token manquant dans le cookie"})
		c.Abort()
		return nil, err
	}

    claims, err := ParseJWT(tokenString)
    if err != nil {
        return nil, fmt.Errorf("token invalide ou expiré: %v", err)
    }
    return claims, nil
}

func ExtractUserClaimsFromToken(token string) (*Claims, error) {
    token = strings.TrimPrefix(token, "Bearer ")
    claims, err := ParseJWT(token)
    if err != nil {
        return nil, fmt.Errorf("token invalide ou expiré: %v", err)
    }
    return claims, nil
}


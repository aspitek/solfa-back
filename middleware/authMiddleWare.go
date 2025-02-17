package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"solfa-back/lib"
)

// AuthMiddleware vérifie la présence et la validité du token JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := lib.ExtractUserClaims(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Accès non autorisé: " + err.Error()})
			c.Abort()
			return
		}

		// Stocke les claims dans le contexte pour les handlers suivants
		c.Set("userClaims", claims)

		// Passe à l'handler suivant
		c.Next()
	}
}

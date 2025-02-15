package handlers

import (
	"github.com/gin-gonic/gin"
)

func GetSolfa(c *gin.Context) {
	c.JSON(200, gin.H{"message": "solfa"})
}

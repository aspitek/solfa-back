package routes

import (
	"github.com/gin-gonic/gin"
	"solfa-back/handlers"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/solfa", handlers.GetSolfa)
}

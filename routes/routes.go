package routes

import (
	"github.com/gin-gonic/gin"
	"solfa-back/handlers"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/solfa", handlers.GetSolfa)
	r.POST("/signup", handlers.SignupHandler)
	r.GET("/verify", handlers.VerifyEmailHandler)
	r.POST("/login", handlers.LoginHandler)
	r.POST("/logout", handlers.LogoutHandler)
}

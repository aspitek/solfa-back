package routes

import (
	"github.com/gin-gonic/gin"
	"solfa-back/handlers"
	"solfa-back/middleware"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/solfa", handlers.GetSolfa)
	r.POST("/signup", handlers.SignupHandler)
	r.GET("/verify", handlers.VerifyEmailHandler)
	r.POST("/login", handlers.LoginHandler)
	r.POST("/logout", handlers.LogoutHandler)
	r.GET("/search", handlers.SearchPartitionsHandler)

	adminGroup := r.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware())
	{
		adminGroup.GET("/me", middleware.AuthMiddleware(), handlers.GetCurrentUser)
		adminGroup.PUT("/me", middleware.AuthMiddleware(), handlers.UpdateCurrentUser)
		adminGroup.GET("/users", middleware.AuthMiddleware(), handlers.GetUserByID)
		adminGroup.POST("/upload", middleware.AuthMiddleware(), handlers.UploadPartitionHandler)
		adminGroup.GET("/validate", middleware.AuthMiddleware(), handlers.ValidatePartitionHandler)
	}
}

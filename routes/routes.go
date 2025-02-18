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
	r.GET("/me", middleware.AuthMiddleware(), handlers.GetCurrentUser)
	r.PUT("/me", middleware.AuthMiddleware(), handlers.UpdateCurrentUser)
	r.GET("/users/:id", middleware.AuthMiddleware(), handlers.GetUserByID)
	r.POST("/upload", middleware.AuthMiddleware(), handlers.UploadPartitionHandler)
	r.POST("/validate", middleware.AuthMiddleware(), handlers.ValidatePartitionHandler)
	r.GET("/search", handlers.SearchPartitionsHandler)
}

package main

import (
	"solfa-back/lib"
	"solfa-back/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	lib.InitDB()
	lib.InitES()
	lib.InitMC()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{
			"http://localhost:3000",
			"http://srv598321.hstgr.cloud",
			"https://srv598321.hstgr.cloud",
			"http://147.79.114.72:32042",
		}, // Permettre uniquement les requêtes de ce domaine
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"}, // Méthodes autorisées
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"}, // En-têtes autorisés
		AllowCredentials: true, // Autoriser les informations d'identification (cookies, etc.)
	}))

	routes.SetupRoutes(r)

	r.Run(":8080")
}

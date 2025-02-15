package main

import "github.com/gin-gonic/gin"
import "github.com/joho/godotenv"
import "solfa-back/lib"
import "solfa-back/routes"


func main() {
	godotenv.Load()
	lib.InitDB()
	lib.InitES()

	r := gin.Default()

	routes.SetupRoutes(r)
	
	r.Run(":8080")
}



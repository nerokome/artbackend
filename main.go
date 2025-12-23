package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/routes"
)

func main() {
	// 1️⃣ Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 2️⃣ Connect to MongoDB (FAIL FAST)
	database.ConnectMongo()

	// 3️⃣ Start Gin
	r := gin.Default()

	// 4️⃣ Health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// 5️⃣ Register routes
	routes.AuthRoutes(r)

	// 6️⃣ Run server
	if err := r.Run(":5005"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

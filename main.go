package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/nerokome/artfolio-backend/config"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/routes"
)

func main() {
	
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config.InitCloudinary()

	database.ConnectMongo()

	
	r := gin.Default()

	
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	
	routes.AuthRoutes(r)       
	routes.ArtworkRoutes(r)    

	if err := r.Run(":5005"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

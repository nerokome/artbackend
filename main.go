package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
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

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"https://artwork-phi-swart.vercel.app",
		},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"X-Visitor-ID",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	routes.AuthRoutes(r)
	routes.ProfileRoutes(r)
	routes.SettingsRoutes(r)
	routes.ArtworkRoutes(r)
	routes.AnalyticsRoutes(r)
	routes.PublicPortfolioRoutes(r)
	routes.SocialRoutes(r)

	if err := r.Run(":5005"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

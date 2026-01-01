package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func AuthRoutes(router *gin.Engine) {
	auth := router.Group("/auth")

	auth.Use(middleware.RateLimiter(0.5, 2))
	{
		auth.POST("/signup", controllers.Signup)
		auth.POST("/login", controllers.Login)
		auth.POST("/logout", controllers.Logout)
	}
}

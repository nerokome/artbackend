package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func PortfolioRoutes(r *gin.Engine) {
	auth := r.Group("/api")
	auth.Use(middleware.Authenticate())
	{
		auth.POST("/portfolios", controllers.CreatePortfolio)
		auth.GET("/portfolios/me", controllers.GetMyPortfolios)
	}

	r.GET("/api/portfolios/:id", controllers.GetPublicPortfolio)
}

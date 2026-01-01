package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func AnalyticsRoutes(router *gin.Engine) {
	analytics := router.Group("/analytics")
	{
		analytics.GET("/overview", middleware.Authenticate(), controllers.GetAnalyticsOverview)
		analytics.GET("/views-over-time", middleware.Authenticate(), controllers.GetViewsOverTime)
		analytics.GET("/most-viewed", middleware.Authenticate(), controllers.GetMostViewedArtworks)
		analytics.GET("/engagement-split", middleware.Authenticate(), controllers.GetEngagementSplit)
		analytics.POST("/log-view/:artworkId", controllers.LogView)
	}
}

package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func AnalyticsRoutes(router *gin.Engine) {
	analytics := router.Group("/analytics")
	{
		// Overview: 5 requests/sec, burst 10
		analytics.GET(
			"/overview",
			middleware.Authenticate(),
			middleware.RateLimiter(5, 10),
			controllers.GetAnalyticsOverview,
		)

		// Views over time: 5 requests/sec, burst 10
		analytics.GET(
			"/views-over-time",
			middleware.Authenticate(),
			middleware.RateLimiter(5, 10),
			controllers.GetViewsOverTime,
		)

		// Most viewed artworks: 5 requests/sec, burst 10
		analytics.GET(
			"/most-viewed",
			middleware.Authenticate(),
			middleware.RateLimiter(5, 10),
			controllers.GetMostViewedArtworks,
		)

		// Engagement split: 5 requests/sec, burst 10
		analytics.GET(
			"/engagement-split",
			middleware.Authenticate(),
			middleware.RateLimiter(5, 10),
			controllers.GetEngagementSplit,
		)

		analytics.POST(
			"/log-view/:artworkId",
			middleware.RateLimiter(10, 20),
			controllers.LogView,
		)
	}
}

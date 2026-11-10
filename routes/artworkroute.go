package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func ArtworkRoutes(router *gin.Engine) {

	artworks := router.Group("/artworks")

	artworks.POST(
		"/upload",
		middleware.Authenticate(),
		middleware.RateLimiter(0.2, 1),
		middleware.UploadMiddleware(10, []string{"image/"}),
		controllers.UploadArtwork,
	)

	artworks.GET(
		"/public",
		middleware.RateLimiter(2, 5),
		controllers.GetPublicArtworks,
	)

	artworks.GET(
		"/popular",
		middleware.RateLimiter(2, 5),
		controllers.GetPopularArtworks,
	)

	artworks.GET(
		"/:id",
		middleware.RateLimiter(1, 5),
		controllers.GetArtworkAndCountView,
	)

	artworks.GET(
		"/mine",
		middleware.Authenticate(),
		middleware.RateLimiter(1, 3),
		controllers.GetMyArtworks,
	)

	likes := artworks.Group("/:id/like")
	likes.GET("", middleware.RateLimiter(10, 30), controllers.GetArtworkLikeStatus)
	likes.POST("", middleware.RateLimiter(2, 10), controllers.LikeArtwork)
	likes.DELETE("", middleware.RateLimiter(2, 10), controllers.UnlikeArtwork)

	artworks.DELETE(
		"/:id",
		middleware.Authenticate(),
		middleware.RateLimiter(0.3, 1),
		controllers.DeleteArtwork,
	)
}

func PublicPortfolioRoutes(router *gin.Engine) {

	portfolio := router.Group("/portfolio")

	portfolio.GET(
		"/artists",
		middleware.Authenticate(),
		middleware.RateLimiter(1, 5),
		controllers.GetPublicArtists,
	)
	portfolio.GET(
		"/id/:id",
		middleware.RateLimiter(1, 5),
		controllers.GetPublicPortfolioByID,
	)
	portfolio.GET(
		"/:name",
		middleware.RateLimiter(1, 5),
		controllers.GetPublicPortfolioByName,
	)
}

package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func ArtworkRoutes(router *gin.Engine) {

	artworks := router.Group("/artworks")

	// 🔒 Upload: expensive + sensitive
	artworks.POST(
		"/upload",
		middleware.Authenticate(),
		middleware.RateLimiter(0.2, 1), // 1 upload every ~5s
		middleware.UploadMiddleware(10, []string{"image/"}),
		controllers.UploadArtwork,
	)

	// 🌍 Public list (light limit)
	artworks.GET(
		"/public",
		middleware.RateLimiter(2, 5),
		controllers.GetPublicArtworks,
	)

	// 👁️ View + analytics (VERY IMPORTANT)
	artworks.GET(
		"/:id",
		middleware.RateLimiter(1, 5),
		controllers.GetArtworkAndCountView,
	)

	// 👤 User’s own artworks
	artworks.GET(
		"/mine",
		middleware.Authenticate(),
		middleware.RateLimiter(1, 3),
		controllers.GetMyArtworks,
	)

	// 🗑️ Delete artwork (destructive)
	artworks.DELETE(
		"/:id",
		middleware.Authenticate(),
		middleware.RateLimiter(0.3, 1), // slow, deliberate deletes
		controllers.DeleteArtwork,
	)
}

func PublicPortfolioRoutes(router *gin.Engine) {

	portfolio := router.Group("/portfolio")

	portfolio.GET(
		"/:name",
		middleware.RateLimiter(1, 5),
		controllers.GetPublicPortfolioByName,
	)
}

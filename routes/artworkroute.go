package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func ArtworkRoutes(router *gin.Engine) {

	artworks := router.Group("/artworks")
	{

		artworks.POST(
			"/upload",
			middleware.Authenticate(),
			middleware.UploadMiddleware(10, []string{"image/"}),
			controllers.UploadArtwork,
		)

		artworks.GET(
			"/public",
			controllers.GetPublicArtworks,
		)

		artworks.GET(
			"/:id",
			controllers.GetArtworkAndCountView,
		)
		artworks.GET(
			"/mine",
			middleware.Authenticate(),
			controllers.GetMyArtworks,
		)

	}
	artworks.DELETE("/:id", middleware.Authenticate(), controllers.DeleteArtwork)

}
func PublicPortfolioRoutes(router *gin.Engine) {

	portfolio := router.Group("/portfolio")
	{

		portfolio.GET(
			"/:name",
			controllers.GetPublicPortfolioByName,
		)
	}
}

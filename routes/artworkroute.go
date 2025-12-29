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
	}
}

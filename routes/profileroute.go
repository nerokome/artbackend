package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func ProfileRoutes(router *gin.Engine) {
	profile := router.Group("/profile")
	profile.Use(middleware.Authenticate())
	profile.GET("", controllers.GetProfile)
	profile.PATCH("", controllers.UpdateProfile)
	profile.PUT("/picture", middleware.RateLimiter(0.2, 1), middleware.UploadMiddleware(5, []string{"image/"}), controllers.UpdateProfilePicture)
}

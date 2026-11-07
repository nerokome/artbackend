package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func SettingsRoutes(router *gin.Engine) {
	settings := router.Group("/settings")
	settings.Use(middleware.Authenticate())
	settings.GET("/account", controllers.GetAccountSettings)
	settings.PATCH("/password", middleware.RateLimiter(0.2, 1), controllers.ChangeAccountPassword)
}

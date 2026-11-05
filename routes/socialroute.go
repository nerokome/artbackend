package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/controllers"
	"github.com/nerokome/artfolio-backend/middleware"
)

func SocialRoutes(router *gin.Engine) {
	notifications := router.Group("/notifications")
	notifications.Use(middleware.Authenticate())
	notifications.GET("", controllers.GetNotifications)
	notifications.PATCH("/:id/read", controllers.MarkNotificationRead)

	follows := router.Group("/artists/:artistId/follow")
	follows.Use(middleware.Authenticate())
	follows.GET("", controllers.GetArtistFollowStatus)
	follows.POST("", controllers.FollowArtist)
	follows.DELETE("", controllers.UnfollowArtist)

	users := router.Group("/users/:userId")
	users.Use(middleware.Authenticate())
	users.GET("/following", controllers.GetFollowingArtists)
	users.GET("/followers", controllers.GetArtistFollowers)

	chat := router.Group("/chat")
	chat.Use(middleware.Authenticate())
	chat.GET("/:artistId/messages", controllers.GetChatHistory)
	// Browsers cannot attach Authorization headers to WebSocket handshakes, so
	// the WebSocket handler validates the same JWT from the token query param.
	router.GET("/chat/ws/:artistId", controllers.ArtistChatWebSocket)
}

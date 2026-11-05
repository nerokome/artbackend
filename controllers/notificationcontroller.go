package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func createNotification(ctx context.Context, recipientID, actorID primitive.ObjectID, kind, reference, content string) {
	_, err := database.Collection("notifications").InsertOne(ctx, bson.M{
		"recipientId": recipientID,
		"actorId":     actorID,
		"type":        kind,
		"reference":   reference,
		"content":     content,
		"isRead":      false,
		"createdAt":   time.Now().UTC(),
	})
	if err != nil { /* Notification failure must not undo a follow or message. */
	}
}

func GetNotifications(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	limit := int64(50)
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	collection := database.Collection("notifications")
	cur, err := collection.Find(ctx, bson.M{"recipientId": userID}, options.Find().SetSort(bson.M{"createdAt": -1}).SetLimit(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch notifications"})
		return
	}
	defer cur.Close(ctx)
	var notifications []bson.M
	if err := cur.All(ctx, &notifications); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse notifications"})
		return
	}
	if notifications == nil {
		notifications = []bson.M{}
	}
	unread, err := collection.CountDocuments(ctx, bson.M{"recipientId": userID, "isRead": false})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count unread notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(notifications), "unreadCount": unread, "notifications": notifications})
}

func MarkNotificationRead(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	notificationID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	result, err := database.Collection("notifications").UpdateOne(ctx, bson.M{"_id": notificationID, "recipientId": userID}, bson.M{"$set": bson.M{"isRead": true, "readAt": time.Now().UTC()}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": notificationID.Hex(), "isRead": true})
}

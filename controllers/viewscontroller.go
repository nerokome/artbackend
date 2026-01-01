package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LogView logs a new view event for a specific artwork
func LogView(c *gin.Context) {
	viewCollection := database.Collection("view_events")
	artworkCollection := database.Collection("artworks")

	artworkIDStr := c.Param("artworkId")

	// 1. Explicit check for common frontend failure strings
	if artworkIDStr == "" || artworkIDStr == "undefined" || artworkIDStr == "null" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid artworkId is required"})
		return
	}

	// 2. Convert string to ObjectID
	objID, err := primitive.ObjectIDFromHex(artworkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 3. Verify artwork exists before logging
	count, err := artworkCollection.CountDocuments(ctx, bson.M{"_id": objID})
	if err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artwork not found"})
		return
	}

	// 4. Create the view event
	view := bson.M{
		"artworkId": objID,
		"userId":    nil, // Expand this later if you add Auth
		"createdAt": time.Now(),
	}

	_, err = viewCollection.InsertOne(ctx, view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log view"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "View logged successfully"})
}

func GetAnalyticsOverview(c *gin.Context) {
	artworkCollection := database.Collection("artworks")
	viewCollection := database.Collection("view_events")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	totalArtworks, _ := artworkCollection.CountDocuments(ctx, bson.M{})
	totalViews, _ := viewCollection.CountDocuments(ctx, bson.M{})

	// Aggregate viewer split (Authenticated vs Public)
	cursor, err := viewCollection.Aggregate(ctx, bson.A{
		bson.M{
			"$group": bson.M{
				"_id": bson.M{
					"$cond": bson.A{
						bson.M{"$ifNull": bson.A{"$userId", false}},
						"authenticated",
						"public",
					},
				},
				"count": bson.M{"$sum": 1},
			},
		},
	})

	var viewerSplit []bson.M
	if err == nil {
		cursor.All(ctx, &viewerSplit)
	}

	c.JSON(http.StatusOK, gin.H{
		"totalArtworks": totalArtworks,
		"totalViews":    totalViews,
		"viewerSplit":   viewerSplit,
	})
}

func GetViewsOverTime(c *gin.Context) {
	viewCollection := database.Collection("view_events")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Last 7 days
	start := time.Now().AddDate(0, 0, -6)

	cursor, err := viewCollection.Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{"createdAt": bson.M{"$gte": start}}},
		bson.M{
			"$group": bson.M{
				"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"}},
				"views": bson.M{"$sum": 1},
			},
		},
		bson.M{"$sort": bson.M{"_id": 1}},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aggregation failed"})
		return
	}

	var result []bson.M
	cursor.All(ctx, &result)
	c.JSON(http.StatusOK, result)
}

func GetMostViewedArtworks(c *gin.Context) {
	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find public artworks and sort by the 'views' field
	opts := options.Find().SetSort(bson.M{"views": -1})
	cursor, err := collection.Find(ctx, bson.M{"isPublic": true}, opts)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}

	var artworks []bson.M
	cursor.All(ctx, &artworks)
	c.JSON(http.StatusOK, artworks)
}

func GetEngagementSplit(c *gin.Context) {
	viewCollection := database.Collection("view_events")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := viewCollection.Aggregate(ctx, bson.A{
		bson.M{
			"$group": bson.M{
				"_id":   "$artworkId",
				"views": bson.M{"$sum": 1},
			},
		},
		bson.M{"$sort": bson.M{"views": -1}},
		bson.M{"$limit": 5},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aggregation failed"})
		return
	}

	var result []bson.M
	cursor.All(ctx, &result)
	c.JSON(http.StatusOK, result)
}

package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetAnalyticsOverview(c *gin.Context) {
	artworkCollection := database.Collection("artworks")
	viewCollection := database.Collection("view_events")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	totalArtworks, err := artworkCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count artworks"})
		return
	}

	totalViews, err := viewCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count views"})
		return
	}

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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics failed"})
		return
	}

	var viewerSplit []bson.M
	if err := cursor.All(ctx, &viewerSplit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics failed"})
		return
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

	start := time.Now().AddDate(0, 0, -6)

	cursor, err := viewCollection.Aggregate(ctx, bson.A{
		bson.M{
			"$match": bson.M{
				"createdAt": bson.M{"$gte": start},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   "$createdAt",
					},
				},
				"views": bson.M{"$sum": 1},
			},
		},
		bson.M{"$sort": bson.M{"_id": 1}},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics failed"})
		return
	}

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func GetMostViewedArtworks(c *gin.Context) {
	collection := database.Collection("artworks")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{"isPublic": true}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch failed"})
		return
	}

	var artworks []bson.M
	if err := cursor.All(ctx, &artworks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch failed"})
		return
	}

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
		bson.M{"$limit": 3},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics failed"})
		return
	}

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}
func LogView(c *gin.Context) {
	viewCollection := database.Collection("view_events")
	artworkCollection := database.Collection("artworks")

	artworkID := c.Param("artworkId")
	if artworkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artworkId required"})
		return
	}

	// Convert string to ObjectID
	objID, err := primitive.ObjectIDFromHex(artworkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artworkId"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if artwork exists
	count, err := artworkCollection.CountDocuments(ctx, bson.M{"_id": objID})
	if err != nil || count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "artwork not found"})
		return
	}

	// Log the view
	view := bson.M{
		"artworkId": objID,
		"userId":    nil,
		"createdAt": time.Now(),
	}

	_, err = viewCollection.InsertOne(ctx, view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log view"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "view logged"})
}

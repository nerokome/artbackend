package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Helper to get authenticated user ID
func getUserID(c *gin.Context) (primitive.ObjectID, bool) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return primitive.NilObjectID, false
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return primitive.NilObjectID, false
	}
	return userID, true
}

// --- Log a view (optional: associate with logged-in viewer) ---
func LogView(c *gin.Context) {
	viewCollection := database.Collection("view_events")
	artworkCollection := database.Collection("artworks")

	artworkID := c.Param("artworkId")
	if artworkID == "" || artworkID == "undefined" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artworkId required"})
		return
	}

	objID, err := primitive.ObjectIDFromHex(artworkID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artworkId"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var viewerID *primitive.ObjectID
	if userIDStr, exists := c.Get("user_id"); exists {
		if id, err := primitive.ObjectIDFromHex(userIDStr.(string)); err == nil {
			viewerID = &id
		}
	}

	view := bson.M{
		"artworkId": objID,
		"userId":    viewerID, 
		"createdAt": time.Now(),
	}

	_, err = viewCollection.InsertOne(ctx, view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log view event"})
		return
	}

	_, err = artworkCollection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$inc": bson.M{"views": 1}},
	)
	if err != nil {
		fmt.Println("Failed to increment view counter:", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "view logged and counter incremented"})
}


func GetAnalyticsOverview(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	artworkCollection := database.Collection("artworks")
	viewCollection := database.Collection("view_events")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	totalArtworks, _ := artworkCollection.CountDocuments(ctx, bson.M{"userId": userID})
	totalViews, _ := viewCollection.CountDocuments(ctx, bson.M{"artworkId": bson.M{"$in": artworkIDsByUser(ctx, userID)}})

	cursor, err := viewCollection.Aggregate(ctx, bson.A{
		bson.M{
			"$match": bson.M{"artworkId": bson.M{"$in": artworkIDsByUser(ctx, userID)}},
		},
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
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	viewCollection := database.Collection("view_events")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now().AddDate(0, 0, -6)

	cursor, err := viewCollection.Aggregate(ctx, bson.A{
		bson.M{
			"$match": bson.M{
				"artworkId": bson.M{"$in": artworkIDsByUser(ctx, userID)},
				"createdAt": bson.M{"$gte": start},
			},
		},
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

// --- Most viewed artworks (per-user) ---
func GetMostViewedArtworks(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"views": -1})
	cursor, err := collection.Find(ctx, bson.M{"isPublic": true, "userId": userID}, opts)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fetch failed"})
		return
	}

	var artworks []bson.M
	cursor.All(ctx, &artworks)
	c.JSON(http.StatusOK, artworks)
}

// --- Engagement split (per-user) ---
func GetEngagementSplit(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	viewCollection := database.Collection("view_events")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := bson.A{
		bson.M{
			"$match": bson.M{"artworkId": bson.M{"$in": artworkIDsByUser(ctx, userID)}},
		},
		bson.M{
			"$group": bson.M{
				"_id":   "$artworkId",
				"views": bson.M{"$sum": 1},
			},
		},
		bson.M{
			"$lookup": bson.M{
				"from":         "artworks",
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "artwork",
			},
		},
		bson.M{"$unwind": "$artwork"},
		bson.M{
			"$project": bson.M{
				"_id":   1,
				"title": "$artwork.title",
				"views": 1,
			},
		},
		bson.M{"$sort": bson.M{"views": -1}},
		bson.M{"$limit": 5},
	}

	cursor, err := viewCollection.Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "aggregation failed"})
		return
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse result"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// --- Helper: fetch artwork IDs for a user ---
func artworkIDsByUser(ctx context.Context, userID primitive.ObjectID) []primitive.ObjectID {
	collection := database.Collection("artworks")
	cursor, err := collection.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return []primitive.ObjectID{}
	}
	defer cursor.Close(ctx)

	var artworks []struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	cursor.All(ctx, &artworks)

	ids := make([]primitive.ObjectID, len(artworks))
	for i, art := range artworks {
		ids[i] = art.ID
	}
	return ids
}

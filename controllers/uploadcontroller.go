package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/config"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UploadArtwork(c *gin.Context) {
	title := c.PostForm("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "file open failed"})
		return
	}
	defer src.Close()

	if config.Cloudinary == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cloudinary not initialized"})
		return
	}

	params := uploader.UploadParams{
		Folder: "artfolio",
	}

	result, err := config.Cloudinary.Upload.Upload(
		context.Background(),
		src,
		params,
	)
	if err != nil {
		fmt.Println("Cloudinary upload error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	artwork := models.Artwork{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Title:     title,
		Slug:      title,
		URL:       result.SecureURL,
		PublicID:  result.PublicID,
		Views:     0,
		IsPublic:  true,
		CreatedAt: time.Now(),
	}

	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := collection.InsertOne(ctx, artwork); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db save failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "upload successful",
		"artwork": artwork,
	})
}

func GetPublicArtworks(c *gin.Context) {
	collection := database.Collection("artworks")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"createdAt": -1})

	cursor, err := collection.Find(
		ctx,
		bson.M{"isPublic": true},
		opts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artworks"})
		return
	}
	defer cursor.Close(ctx)

	var artworks []models.Artwork
	if err := cursor.All(ctx, &artworks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse artworks"})
		return
	}

	c.JSON(http.StatusOK, artworks)
}

func GetArtworkAndCountView(c *gin.Context) {
	id := c.Param("id")

	artworkID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artwork id"})
		return
	}

	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var artwork models.Artwork
	err = collection.FindOne(
		ctx,
		bson.M{
			"_id":      artworkID,
			"isPublic": true,
		},
	).Decode(&artwork)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "artwork not found"})
		return
	}

	// Increment view count (public viewers)
	go collection.UpdateOne(
		context.Background(),
		bson.M{"_id": artworkID},
		bson.M{"$inc": bson.M{"views": 1}},
	)

	c.JSON(http.StatusOK, artwork)
}
func GetMyArtworks(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userIDHex, ok := userIDValue.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id format",
		})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	
	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	
	opts := options.Find().
		SetSort(bson.M{"createdAt": -1})

	cursor, err := collection.Find(
		ctx,
		bson.M{
			"userId": userID,
		},
		opts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch artworks",
		})
		return
	}
	defer cursor.Close(ctx)

	var artworks []models.Artwork
	if err := cursor.All(ctx, &artworks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to parse artworks",
		})
		return
	}

	
	c.JSON(http.StatusOK, gin.H{
		"count":    len(artworks),
		"artworks": artworks,
	})
}

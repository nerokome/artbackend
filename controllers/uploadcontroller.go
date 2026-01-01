package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
func GetPublicPortfolioByName(c *gin.Context) {
	nameSlug := c.Param("name")

	normalizedName := strings.ReplaceAll(nameSlug, "-", " ")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Find user by full name (case-insensitive)
	userCollection := database.Collection("users")
	var user models.User

	err := userCollection.FindOne(
		ctx,
		bson.M{
			"full_name": bson.M{
				"$regex":   "^" + normalizedName + "$",
				"$options": "i",
			},
		},
	).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "artist not found",
		})
		return
	}

	// 2. Fetch ONLY public artworks
	artworkCollection := database.Collection("artworks")

	cursor, err := artworkCollection.Find(
		ctx,
		bson.M{
			"userId":   user.ID,
			"isPublic": true,
		},
		options.Find().SetSort(bson.M{"createdAt": -1}),
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

	// 3. Public-safe response
	c.JSON(http.StatusOK, gin.H{
		"profile": gin.H{
			"name": user.FullName,
		},
		"count":    len(artworks),
		"artworks": artworks,
	})
}
func DeleteArtwork(c *gin.Context) {
	// 1. Get artwork ID from URL param
	artworkIDStr := c.Param("id")
	if artworkIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artwork ID is required"})
		return
	}

	artworkID, err := primitive.ObjectIDFromHex(artworkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artwork ID"})
		return
	}

	// 2. Get user ID from context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDHex, ok := userIDVal.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	// 3. Find the artwork to verify ownership
	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var artwork models.Artwork
	err = collection.FindOne(ctx, bson.M{"_id": artworkID, "userId": userID}).Decode(&artwork)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "artwork not found or you do not own it"})
		return
	}

	// 4. Delete from Cloudinary
	if config.Cloudinary != nil && artwork.PublicID != "" {
		_, err = config.Cloudinary.Upload.Destroy(context.Background(), uploader.DestroyParams{
			PublicID: artwork.PublicID,
		})
		if err != nil {
			fmt.Println("Cloudinary deletion error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete artwork from cloud storage"})
			return
		}
	}

	// 5. Delete from MongoDB
	_, err = collection.DeleteOne(ctx, bson.M{"_id": artworkID, "userId": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete artwork from database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "artwork deleted successfully",
	})
}

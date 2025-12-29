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
	"go.mongodb.org/mongo-driver/bson/primitive"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "file open failed: " + err.Error()})
		return
	}
	defer src.Close()

	params := uploader.UploadParams{Folder: "artfolio"}

	if config.Cloudinary == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cloudinary not initialized"})
		return
	}

	result, err := config.Cloudinary.Upload.Upload(context.Background(), src, params)
	if err != nil {
		fmt.Println("Cloudinary upload error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed: " + err.Error()})
		return
	}

	if result.SecureURL == "" || result.PublicID == "" {
		fmt.Println("Cloudinary returned empty result:", result)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed: empty URL or public ID"})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id missing in context"})
		return
	}

	userIDHex, ok := userIDStr.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	slug := title

	artwork := models.Artwork{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Title:     title,
		Slug:      slug,
		URL:       result.SecureURL,
		PublicID:  result.PublicID,
		Views:     0,
		IsPublic:  true,
		CreatedAt: time.Now(),
	}

	// 8. Insert into MongoDB
	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := collection.InsertOne(ctx, artwork); err != nil {
		fmt.Println("MongoDB insert error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db save failed: " + err.Error()})
		return
	}

	// 9. Respond success
	c.JSON(http.StatusCreated, gin.H{
		"message": "upload successful",
		"artwork": artwork,
	})
}

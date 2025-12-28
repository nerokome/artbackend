package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/config"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/models"
	"github.com/nerokome/artfolio-backend/utils"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "file open failed"})
		return
	}
	defer src.Close()

	// Watermark
	watermark := c.PostForm("watermark") == "true"
	watermarkText := c.PostForm("watermark_text")

	var transform []string
	if watermark && watermarkText != "" {
		escaped := strings.ReplaceAll(watermarkText, " ", "%20")
		transform = append(transform,
			"l_text:arial_40:"+escaped+",g_south_east,x_20,y_20,opacity_40,co_white",
		)
	}

	params := uploader.UploadParams{
		Folder: "artfolio",
	}

	if len(transform) > 0 {
		params.Transformation = strings.Join(transform, "/")
	}

	// Use Cloudinary client correctly
	result, err := config.Cloudinary.Upload.Upload(
		context.Background(),
		src,
		params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}

	userID := c.MustGet("userId").(primitive.ObjectID)
	slug := utils.GenerateSlug(title)
	seo := utils.GenerateSEO(title, result.SecureURL)

	artwork := models.Artwork{
		ID:       primitive.NewObjectID(),
		UserID:   userID,
		Title:    title,
		Slug:     slug,
		URL:      result.SecureURL,
		PublicID: result.PublicID,
		Views:    0,
		IsPublic: true,
		SEO: models.SEOMetadata{
			Title:       seo["title"].(string),
			Description: seo["description"].(string),
			Image:       seo["image"].(string),
			Keywords:    seo["keywords"].([]string),
		},
		CreatedAt: time.Now(),
	}

	// Dynamic MongoDB collection helper
	collection := database.Collection("artworks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = collection.InsertOne(ctx, artwork)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db save failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "upload successful",
		"artwork": artwork,
	})
}

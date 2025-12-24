package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreatePortfolio(c *gin.Context) {
	userIDStr, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))
	collection := database.Collection("portfolios")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		ImageURL    string   `json:"imageUrl"`
		Tags        []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	portfolio := models.Portfolio{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Title:       input.Title,
		Description: input.Description,
		ImageURL:    input.ImageURL,
		Tags:        input.Tags,
		Views:       0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, portfolio)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create portfolio"})
		return
	}

	c.JSON(http.StatusCreated, portfolio)
}
func GetMyPortfolios(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	collection := database.Collection("portfolios")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch portfolios"})
		return
	}
	defer cursor.Close(ctx)

	var portfolios []models.Portfolio
	if err := cursor.All(ctx, &portfolios); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data parsing failed"})
		return
	}

	c.JSON(http.StatusOK, portfolios)
}
func GetPublicPortfolio(c *gin.Context) {
	id := c.Param("id")
	portfolioID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	collection := database.Collection("portfolios")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var portfolio models.Portfolio
	err = collection.FindOne(ctx, bson.M{"_id": portfolioID}).Decode(&portfolio)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Portfolio not found"})
		return
	}

	// Increment views (fire-and-forget)
	go collection.UpdateOne(
		context.Background(),
		bson.M{"_id": portfolioID},
		bson.M{"$inc": bson.M{"views": 1}},
	)

	c.JSON(http.StatusOK, portfolio)
}

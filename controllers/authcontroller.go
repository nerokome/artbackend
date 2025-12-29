package controllers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// helper function to respond with JSON error
func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
	c.Abort()
}

// SIGNUP
func Signup(c *gin.Context) {
	userCollection := database.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input struct {
		FullName string `json:"fullName" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	count, err := userCollection.CountDocuments(ctx, bson.M{"email": input.Email})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}
	if count > 0 {
		respondError(c, http.StatusConflict, "Email already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Could not hash password")
		return
	}

	user := models.User{
		ID:        primitive.NewObjectID(),
		FullName:  input.FullName,
		Email:     input.Email,
		Password:  string(hashedPassword),
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if _, err := userCollection.InsertOne(ctx, user); err != nil {
		respondError(c, http.StatusInternalServerError, "Could not create user: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Signup successful"})
}

// LOGIN
func Login(c *gin.Context) {
	userCollection := database.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&user); err != nil {
		respondError(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		respondError(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		respondError(c, http.StatusInternalServerError, "JWT secret not configured")
		return
	}

	claims := jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Could not generate token: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": signedToken,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"name":  user.FullName,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

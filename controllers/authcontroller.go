package controllers

import (
	"context"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// -----------------------------
// Helper: Generate JWT
// -----------------------------
func generateJWT(user models.User, duration time.Duration) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", nil
	}
	claims := jwt.MapClaims{
		"userId": user.ID.Hex(),
		"email":  user.Email,
		"role":   user.Role,
		"exp":    time.Now().Add(duration).Unix(),
		"iat":    time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// -----------------------------
// Helper: Slugify
// -----------------------------
func slugify(input string) string {
	s := strings.ToLower(input)
	reg := regexp.MustCompile(`[^\w\s-]`)
	s = reg.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	reg2 := regexp.MustCompile(`-+`)
	s = reg2.ReplaceAllString(s, "-")
	return s
}

// -----------------------------
// SIGNUP
// -----------------------------
func Signup(c *gin.Context) {
	userCollection := database.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure unique index on email and slug
	_, err := userCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.M{"email": 1}, Options: options.Index().SetUnique(true)},
		{Keys: bson.M{"slug": 1}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		log.Println("Unique index creation skipped:", err)
	}

	// Input validation
	var input struct {
		FullName string `json:"fullName" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Password hashing failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Generate slug
	slug := slugify(input.FullName)

	user := models.User{
		ID:        primitive.NewObjectID(),
		FullName:  input.FullName,
		Email:     input.Email,
		Password:  string(hashedPassword),
		Role:      "user",
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = userCollection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Email or username already exists"})
		} else {
			log.Println("Failed to insert user:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	token, err := generateJWT(user, 7*24*time.Hour)
	if err != nil {
		log.Println("JWT generation failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Signup successful",
		"token":   token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"name":  user.FullName,
			"email": user.Email,
			"role":  user.Role,
			"slug":  user.Slug,
		},
	})
}

// -----------------------------
// LOGIN
// -----------------------------
func Login(c *gin.Context) {
	userCollection := database.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	var user models.User
	if err := userCollection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := generateJWT(user, 7*24*time.Hour)
	if err != nil {
		log.Println("JWT generation failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID.Hex(),
			"name":  user.FullName,
			"email": user.Email,
			"role":  user.Role,
			"slug":  user.Slug,
		},
	})
}

// -----------------------------
// LOGOUT
// -----------------------------
func Logout(c *gin.Context) {
	// Stateless JWT — client just deletes token
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// -----------------------------
// GET PUBLIC PORTFOLIO BY SLUG
// -----------------------------

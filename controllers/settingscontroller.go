package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// GetAccountSettings returns the signed-in user's account details without
// exposing the stored password hash.
func GetAccountSettings(c *gin.Context) {
	userID, ok := settingsUserID(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var user struct {
		ID        primitive.ObjectID `bson:"_id" json:"id"`
		FullName  string             `bson:"full_name" json:"fullName"`
		Email     string             `bson:"email" json:"email"`
		Role      string             `bson:"role" json:"role"`
		CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
	}
	if err := database.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account": user})
}

// ChangeAccountPassword verifies the current password before saving a new hash.
func ChangeAccountPassword(c *gin.Context) {
	userID, ok := settingsUserID(c)
	if !ok {
		return
	}
	var input struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "currentPassword and newPassword (at least 6 characters) are required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	users := database.Collection("users")
	var account struct {
		Password string `bson:"password"`
	}
	if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&account); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(input.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure new password"})
		return
	}
	if _, err := users.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{"password": string(hash), "updated_at": time.Now()}}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

func settingsUserID(c *gin.Context) (primitive.ObjectID, bool) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return primitive.NilObjectID, false
	}
	value, ok := userIDStr.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return primitive.NilObjectID, false
	}
	userID, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return primitive.NilObjectID, false
	}
	return userID, true
}

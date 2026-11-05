package controllers

import (
	"context"
	"log"
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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetProfile(c *gin.Context) {
	profile, ok := loadProfile(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

func UpdateProfile(c *gin.Context) {
	userID, ok := profileUserID(c)
	if !ok {
		return
	}
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	userUpdate, err := database.Collection("users").UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{
		"full_name":  name,
		"updated_at": time.Now(),
	}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update name"})
		return
	}
	if userUpdate.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	profiles := database.Collection("profiles")
	if _, err := profiles.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$set":         bson.M{"name": name, "updatedAt": time.Now()},
		"$setOnInsert": bson.M{"userId": userID, "createdAt": time.Now()},
	}, optionsUpsert()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "name changed on account but failed to update profile; please retry"})
		return
	}
	var profile models.Profile
	if err := profiles.FindOne(ctx, bson.M{"_id": userID}).Decode(&profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated profile"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

func UpdateProfilePicture(c *gin.Context) {
	userID, ok := profileUserID(c)
	if !ok {
		return
	}
	if config.Cloudinary == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cloudinary not initialized"})
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	profile, ok := loadProfileWithContext(c, ctx, userID)
	if !ok {
		return
	}
	upload, err := config.Cloudinary.Upload.Upload(ctx, src, uploader.UploadParams{
		Folder:   "artfolio/profiles",
		PublicID: userID.Hex() + "_" + primitive.NewObjectID().Hex(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "profile picture upload failed"})
		return
	}

	profiles := database.Collection("profiles")
	_, err = profiles.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{
		"userId":                 userID,
		"profilePictureUrl":      upload.SecureURL,
		"profilePicturePublicId": upload.PublicID,
		"updatedAt":              time.Now(),
	}, "$setOnInsert": bson.M{"name": profile.Name, "createdAt": time.Now()}}, optionsUpsert())
	if err != nil {
		_, _ = config.Cloudinary.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: upload.PublicID})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save profile picture"})
		return
	}
	if profile.ProfilePicturePublicID != "" {
		if _, err := config.Cloudinary.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: profile.ProfilePicturePublicID}); err != nil {
			log.Printf("failed to remove previous profile picture for user %s: %v", userID.Hex(), err)
		}
	}

	profile.ProfilePictureURL = upload.SecureURL
	profile.ProfilePicturePublicID = upload.PublicID
	profile.UpdatedAt = time.Now()
	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

func loadProfile(c *gin.Context) (models.Profile, bool) {
	userID, ok := profileUserID(c)
	if !ok {
		return models.Profile{}, false
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	return loadProfileWithContext(c, ctx, userID)
}

func loadProfileWithContext(c *gin.Context, ctx context.Context, userID primitive.ObjectID) (models.Profile, bool) {
	profiles := database.Collection("profiles")
	var profile models.Profile
	err := profiles.FindOne(ctx, bson.M{"_id": userID}).Decode(&profile)
	if err == nil {
		return profile, true
	}
	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile"})
		return models.Profile{}, false
	}
	var user models.User
	if err := database.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		}
		return models.Profile{}, false
	}
	now := time.Now()
	profile = models.Profile{ID: userID, UserID: userID, Name: user.FullName, CreatedAt: now, UpdatedAt: now}
	if _, err := profiles.InsertOne(ctx, profile); err != nil {
		// Another request may have created the profile at the same time.
		if err := profiles.FindOne(ctx, bson.M{"_id": userID}).Decode(&profile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create profile"})
			return models.Profile{}, false
		}
	}
	return profile, true
}

func profileUserID(c *gin.Context) (primitive.ObjectID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return primitive.NilObjectID, false
	}
	id, ok := value.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return primitive.NilObjectID, false
	}
	userID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return primitive.NilObjectID, false
	}
	return userID, true
}

// Kept as a helper so both profile writes use the same upsert behavior.
func optionsUpsert() *options.UpdateOptions {
	return options.Update().SetUpsert(true)
}

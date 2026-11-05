package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func FollowArtist(c *gin.Context)   { setArtistFollow(c, true) }
func UnfollowArtist(c *gin.Context) { setArtistFollow(c, false) }

func setArtistFollow(c *gin.Context, follow bool) {
	viewerID, ok := getUserID(c)
	if !ok {
		return
	}
	artistID, err := primitive.ObjectIDFromHex(c.Param("artistId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artist id"})
		return
	}
	if artistID == viewerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you cannot follow yourself"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	if err := database.Collection("users").FindOne(ctx, bson.M{"_id": artistID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find artist"})
		}
		return
	}
	follows := database.Collection("follows")
	key := viewerID.Hex() + ":" + artistID.Hex()
	if follow {
		_, err := follows.InsertOne(ctx, bson.M{"_id": key, "followerId": viewerID, "artistId": artistID, "createdAt": time.Now()})
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to follow artist"})
			return
		}
		if err == nil {
			var follower struct {
				FullName string `bson:"full_name"`
			}
			_ = database.Collection("users").FindOne(ctx, bson.M{"_id": viewerID}).Decode(&follower)
			createNotification(ctx, artistID, viewerID, "follow", "follow", follower.FullName+" started following you.")
		}
	} else {
		_, err := follows.DeleteOne(ctx, bson.M{"_id": key})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unfollow artist"})
			return
		}
	}
	count, err := follows.CountDocuments(ctx, bson.M{"artistId": artistID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count followers"})
		return
	}
	followingCount, err := follows.CountDocuments(ctx, bson.M{"followerId": viewerID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count following"})
		return
	}
	var existing bson.M
	_ = follows.FindOne(ctx, bson.M{"_id": key}).Decode(&existing)
	c.JSON(http.StatusOK, gin.H{"artistId": artistID.Hex(), "isFollowing": existing != nil, "followerCount": count, "followingCount": followingCount})
}

func GetArtistFollowStatus(c *gin.Context) {
	viewerID, ok := getUserID(c)
	if !ok {
		return
	}
	artistID, err := primitive.ObjectIDFromHex(c.Param("artistId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artist id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	if err := database.Collection("users").FindOne(ctx, bson.M{"_id": artistID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find artist"})
		}
		return
	}
	follows := database.Collection("follows")
	count, err := follows.CountDocuments(ctx, bson.M{"artistId": artistID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count followers"})
		return
	}
	following, err := follows.CountDocuments(ctx, bson.M{"followerId": viewerID, "artistId": artistID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch follow status"})
		return
	}
	followingCount, err := follows.CountDocuments(ctx, bson.M{"followerId": viewerID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count following"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"artistId": artistID.Hex(), "isFollowing": following > 0, "followerCount": count, "followingCount": followingCount})
}

package controllers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var visitorIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func LikeArtwork(c *gin.Context)   { setArtworkLike(c, true) }
func UnlikeArtwork(c *gin.Context) { setArtworkLike(c, false) }

// Likes can be made by signed-in users or anonymous visitors. Signed-in likes
// retain the existing userId:artworkId key; guest likes use a browser UUID.
func artworkLikeActor(c *gin.Context, required bool) (string, primitive.ObjectID, bool, bool) {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth != "" {
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return "", primitive.NilObjectID, false, false
		}
		claims, err := utils.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return "", primitive.NilObjectID, false, false
		}
		userID, _ := claims["user_id"].(string)
		if userID == "" {
			userID, _ = claims["userId"].(string)
		}
		id, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in token"})
			return "", primitive.NilObjectID, false, false
		}
		return id.Hex(), id, true, true
	}
	visitorID := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Visitor-ID")))
	if visitorID == "" {
		if !required {
			return "", primitive.NilObjectID, false, true
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Visitor-ID header with a UUID is required for anonymous likes"})
		return "", primitive.NilObjectID, false, false
	}
	if !visitorIDPattern.MatchString(visitorID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Visitor-ID must be a valid UUID"})
		return "", primitive.NilObjectID, false, false
	}
	return "guest:" + visitorID, primitive.NilObjectID, false, true
}

func setArtworkLike(c *gin.Context, like bool) {
	actorID, userID, authenticated, ok := artworkLikeActor(c, true)
	if !ok {
		return
	}
	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artwork id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	artworks := database.Collection("artworks")
	var artwork struct {
		UserID primitive.ObjectID `bson:"userId"`
		Title  string             `bson:"title"`
	}
	if err := artworks.FindOne(ctx, bson.M{"_id": artworkID, "isPublic": true}).Decode(&artwork); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "artwork not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find artwork"})
		}
		return
	}
	likes := database.Collection("artwork_likes")
	key := actorID + ":" + artworkID.Hex()
	if like {
		doc := bson.M{"_id": key, "actorId": actorID, "artworkId": artworkID, "createdAt": time.Now().UTC()}
		if authenticated {
			doc["userId"] = userID
		}
		_, err := likes.InsertOne(ctx, doc)
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like artwork"})
			return
		}
		if err == nil {
			if _, err := artworks.UpdateOne(ctx, bson.M{"_id": artworkID}, bson.M{"$inc": bson.M{"likes": 1}}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "like saved but count update failed"})
				return
			}
			if authenticated && artwork.UserID != userID {
				var liker struct {
					FullName string `bson:"full_name"`
				}
				_ = database.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&liker)
				createNotification(ctx, artwork.UserID, userID, "like", artworkID.Hex(), liker.FullName+" liked your artwork: "+artwork.Title)
			}
		}
	} else {
		result, err := likes.DeleteOne(ctx, bson.M{"_id": key})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlike artwork"})
			return
		}
		if result.DeletedCount > 0 {
			if _, err := artworks.UpdateOne(ctx, bson.M{"_id": artworkID, "likes": bson.M{"$gt": 0}}, bson.M{"$inc": bson.M{"likes": -1}}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "like removed but count update failed"})
				return
			}
		}
	}
	returnArtworkLikeStatus(c, ctx, artworkID, key)
}

func GetArtworkLikeStatus(c *gin.Context) {
	actorID, _, _, ok := artworkLikeActor(c, false)
	if !ok {
		return
	}
	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artwork id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	returnArtworkLikeStatus(c, ctx, artworkID, actorID+":"+artworkID.Hex())
}

func returnArtworkLikeStatus(c *gin.Context, ctx context.Context, artworkID primitive.ObjectID, key string) {
	var artwork struct {
		Likes int `bson:"likes"`
	}
	if err := database.Collection("artworks").FindOne(ctx, bson.M{"_id": artworkID, "isPublic": true}).Decode(&artwork); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "artwork not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch artwork"})
		}
		return
	}
	var liked int64
	if key != ":"+artworkID.Hex() {
		var err error
		liked, err = database.Collection("artwork_likes").CountDocuments(ctx, bson.M{"_id": key})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch like status"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"artworkId": artworkID.Hex(), "isLiked": liked > 0, "likeCount": artwork.Likes})
}

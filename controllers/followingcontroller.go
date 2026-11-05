package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nerokome/artfolio-backend/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetFollowingArtists(c *gin.Context) { getRelatedArtists(c, "following") }
func GetArtistFollowers(c *gin.Context)  { getRelatedArtists(c, "followers") }

func getRelatedArtists(c *gin.Context, relation string) {
	if _, ok := getUserID(c); !ok {
		return
	}
	userID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	var match, join bson.M
	if relation == "following" {
		match = bson.M{"followerId": userID}
		join = bson.M{"from": "users", "localField": "artistId", "foreignField": "_id", "as": "relatedUser"}
	} else {
		match = bson.M{"artistId": userID}
		join = bson.M{"from": "users", "localField": "followerId", "foreignField": "_id", "as": "relatedUser"}
	}
	pipeline := bson.A{
		bson.M{"$match": match},
		bson.M{"$lookup": join},
		bson.M{"$unwind": "$relatedUser"},
		bson.M{"$lookup": bson.M{"from": "profiles", "localField": "relatedUser._id", "foreignField": "_id", "as": "profile"}},
		bson.M{"$unwind": bson.M{"path": "$profile", "preserveNullAndEmptyArrays": true}},
		bson.M{"$project": bson.M{
			"_id": 0, "id": "$relatedUser._id",
			"name":              bson.M{"$ifNull": bson.A{"$profile.name", "$relatedUser.full_name"}},
			"profilePictureUrl": bson.M{"$ifNull": bson.A{"$profile.profilePictureUrl", ""}},
		}},
		bson.M{"$sort": bson.M{"name": 1}},
	}
	cur, err := database.Collection("follows").Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch " + relation})
		return
	}
	defer cur.Close(ctx)
	var artists []bson.M
	if err := cur.All(ctx, &artists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse " + relation})
		return
	}
	if artists == nil {
		artists = []bson.M{}
	}
	c.JSON(http.StatusOK, gin.H{"userId": userID.Hex(), relation + "Count": len(artists), relation: artists})
}

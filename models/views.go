package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ViewEvent represents a single view of an artwork
type ViewEvent struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ArtworkID primitive.ObjectID  `bson:"artworkId" json:"artworkId"`
	UserID    *primitive.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	CreatedAt time.Time           `bson:"createdAt" json:"createdAt"`
}

// Artwork represents the artwork document in MongoDB
type Artworker struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title     string             `bson:"title" json:"title"`
	URL       string             `bson:"url" json:"url"`
	Views     int                `bson:"views" json:"views"`
	IsPublic  bool               `bson:"isPublic" json:"isPublic"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

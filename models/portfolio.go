package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Portfolio struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	ImageURL    string             `bson:"imageUrl" json:"imageUrl"`
	Tags        []string           `bson:"tags" json:"tags"`
	Views       int64              `bson:"views" json:"views"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

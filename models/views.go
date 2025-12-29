package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ViewEvent struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ArtworkID primitive.ObjectID  `bson:"artworkId" json:"artworkId"`
	UserID    *primitive.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	CreatedAt time.Time           `bson:"createdAt" json:"createdAt"`
}

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Profile stores the public-facing name and picture for one user.
type Profile struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID                 primitive.ObjectID `bson:"userId" json:"userId"`
	Name                   string             `bson:"name" json:"name"`
	ProfilePictureURL      string             `bson:"profilePictureUrl,omitempty" json:"profilePictureUrl,omitempty"`
	ProfilePicturePublicID string             `bson:"profilePicturePublicId,omitempty" json:"-"`
	CreatedAt              time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt              time.Time          `bson:"updatedAt" json:"updatedAt"`
}

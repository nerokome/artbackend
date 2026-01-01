package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FullName       string             `bson:"full_name" json:"fullName"`
	Email          string             `bson:"email" json:"email"`
	Password       string             `bson:"password,omitempty" json:"-"`
	Role           string             `bson:"role" json:"role"`
	PortfolioViews int                `bson:"portfolioViews" json:"portfolioViews"`
	CreatedAt      time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updatedAt"`
}

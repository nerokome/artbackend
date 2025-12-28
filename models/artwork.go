package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Artwork struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Title     string             `bson:"title" json:"title"`
	Slug      string             `bson:"slug" json:"slug"`
	URL       string             `bson:"url" json:"url"`
	PublicID  string             `bson:"publicId" json:"publicId"`
	Views     int                `bson:"views" json:"views"`
	IsPublic  bool               `bson:"isPublic" json:"isPublic"`
	SEO       SEOMetadata        `bson:"seo" json:"seo"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type SEOMetadata struct {
	Title       string   `bson:"title" json:"title"`
	Description string   `bson:"description" json:"description"`
	Image       string   `bson:"image" json:"image"`
	Keywords    []string `bson:"keywords" json:"keywords"`
}

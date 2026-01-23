package database

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client             *mongo.Client
	once               sync.Once
	databaseName       = "artfolio"
)

func ConnectMongo() {
	once.Do(func() {
		mongoURI := os.Getenv("MONGO_URI")
		if mongoURI == "" {
			log.Fatal("MONGO_URI is not set")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		if err != nil {
			log.Fatal("Mongo connection failed:", err)
		}

		if err := client.Ping(ctx, nil); err != nil {
			log.Fatal("Mongo ping failed:", err)
		}

		Client = client
		log.Println("✅ Connected to MongoDB")
	})
}

// Collection returns a collection reference
func Collection(name string) *mongo.Collection {
	if Client == nil {
		log.Fatal("Mongo client is not initialized. Call ConnectMongo() first.")
	}
	return Client.Database(databaseName).Collection(name)
}

// Optional helper for Users collection
func UserCollection() *mongo.Collection {
	return Collection("users")
}

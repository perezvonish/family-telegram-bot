package database

import (
	"context"
	"fmt"
	"log"
	"perezvonish/health-tracker/internal/shared/config"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(ctx context.Context, cfg config.MongoConfig) (*MongoDB, error) {
	uri := cfg.URI
	if uri == "" {
		if cfg.Username != "" && cfg.Password != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=admin", cfg.Username, cfg.Password, cfg.Host, cfg.Port)
		} else {
			uri = fmt.Sprintf("mongodb://%s:%d", cfg.Host, cfg.Port)
		}
	}

	clientOpts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(10 * time.Second).
		SetConnectTimeout(10 * time.Second)

	var client *mongo.Client
	var err error

	for attempt := 1; attempt <= cfg.ConnectRetryCount; attempt++ {
		log.Printf("Connecting to MongoDB (attempt %d/%d)...", attempt, cfg.ConnectRetryCount)

		client, err = mongo.Connect(ctx, clientOpts)
		if err != nil {
			log.Printf("Failed to connect: %v", err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if err = client.Ping(ctx, readpref.Primary()); err != nil {
			log.Printf("Failed to ping MongoDB: %v", err)
			client.Disconnect(ctx)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		log.Printf("Connected to MongoDB successfully")
		break
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB after %d attempts: %w", cfg.ConnectRetryCount, err)
	}

	return &MongoDB{
		Client:   client,
		Database: client.Database(cfg.DatabaseName),
	}, nil
}

func (m *MongoDB) Close(ctx context.Context) error {
	if m.Client != nil {
		log.Println("Disconnecting from MongoDB...")
		return m.Client.Disconnect(ctx)
	}
	return nil
}

func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

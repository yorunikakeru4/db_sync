package storage

import (
	"context"
	"fmt"
	"time"

	"db_sync/internal/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// InitMongoDB connects to MongoDB using the global MongoConnect config, pings the
// primary to verify connectivity, and returns the configured database.
func InitMongoDB() (*mongo.Client, *mongo.Database, error) {
	cfg := config.MongoConnect

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%s/?authSource=admin",
		cfg.User, cfg.Password, cfg.URL, cfg.Port,
	)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, nil, err
	}

	db := client.Database(cfg.DBName)
	return client, db, nil
}

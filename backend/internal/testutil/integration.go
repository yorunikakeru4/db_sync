// Package testutil provides integration helpers for backend tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// TestDB holds live PostgreSQL and MongoDB connections for backend integration tests.
type TestDB struct {
	PG          *sqlx.DB
	Mongo       *mongo.Database
	mongoClient *mongo.Client
}

// NewTestDB connects to PostgreSQL and MongoDB using env vars.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	pg, err := sqlx.Connect("pgx", postgresURL())
	if err != nil {
		t.Skipf("integration: skipping — cannot connect to PostgreSQL: %v", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURL()))
	if err != nil {
		_ = pg.Close()
		t.Skipf("integration: skipping — cannot connect to MongoDB: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = pg.Close()
		_ = client.Disconnect(context.Background())
		t.Skipf("integration: skipping — MongoDB ping failed: %v", err)
	}

	db := &TestDB{
		PG:          pg,
		Mongo:       client.Database("email_service"),
		mongoClient: client,
	}

	t.Cleanup(func() {
		_ = pg.Close()
		_ = client.Disconnect(context.Background())
	})

	return db
}

// Reset clears PostgreSQL and MongoDB state for a clean test case.
func (db *TestDB) Reset(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	if _, err := db.PG.ExecContext(ctx, "TRUNCATE users_emails, emails, messages, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("backend testutil: truncate postgres: %v", err)
	}
	if err := db.Mongo.Collection("users").Drop(ctx); err != nil && !isMongoNamespaceMissing(err) {
		t.Fatalf("backend testutil: drop mongo users: %v", err)
	}
}

func postgresURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getEnv("POSTGRES_USER", "db_user"),
		getEnv("POSTGRES_PASSWORD", "db_password"),
		getEnv("TEST_POSTGRES_HOST", "localhost"),
		getEnv("TEST_POSTGRES_PORT", "5432"),
		getEnv("POSTGRES_DB", "db_name"),
	)
}

func mongoURL() string {
	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%s/?authSource=admin",
		getEnv("MONGO_INITDB_ROOT_USERNAME", "mongo_user"),
		getEnv("MONGO_INITDB_ROOT_PASSWORD", "mongo_password"),
		getEnv("TEST_MONGO_HOST", "localhost"),
		getEnv("TEST_MONGO_PORT", "27017"),
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func isMongoNamespaceMissing(err error) bool {
	cmdErr, ok := err.(mongo.CommandError)
	return ok && cmdErr.Code == 26
}

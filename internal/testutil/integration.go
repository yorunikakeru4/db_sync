// Package testutil provides helpers for integration tests.
// It connects to real PostgreSQL and MongoDB instances.
//
// Required env vars (defaults match .env.example / docker-compose):
//
//	TEST_POSTGRES_HOST         (default: localhost)
//	TEST_POSTGRES_PORT         (default: 5432)
//	POSTGRES_USER              (default: db_user)
//	POSTGRES_PASSWORD          (default: db_password)
//	POSTGRES_DB                (default: db_name)
//	TEST_MONGO_HOST            (default: localhost)
//	TEST_MONGO_PORT            (default: 27017)
//	MONGO_INITDB_ROOT_USERNAME (default: mongo_user)
//	MONGO_INITDB_ROOT_PASSWORD (default: mongo_password)
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

// TestDB holds live connections to PostgreSQL and MongoDB for integration tests.
type TestDB struct {
	PG          *sqlx.DB
	Mongo       *mongo.Database
	mongoClient *mongo.Client
}

// NewTestDB connects to PostgreSQL and MongoDB using env vars.
// The test is skipped (not failed) if either connection cannot be established,
// so the suite still passes in CI environments without the stack running.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	pg, err := sqlx.Connect("pgx", postgresURL())
	if err != nil {
		t.Skipf("integration: skipping — cannot connect to PostgreSQL: %v", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURL()))
	if err != nil {
		pg.Close()
		t.Skipf("integration: skipping — cannot connect to MongoDB: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		pg.Close()
		_ = client.Disconnect(context.Background())
		t.Skipf("integration: skipping — MongoDB ping failed: %v", err)
	}

	db := &TestDB{
		PG:          pg,
		Mongo:       client.Database("email_service"),
		mongoClient: client,
	}

	t.Cleanup(func() {
		pg.Close()
		_ = client.Disconnect(context.Background())
	})

	return db
}

// Reset wipes all test data: truncates the PostgreSQL users table (cascades to
// messages and emails) and drops the MongoDB users collection.
// Call at the start of each test for a clean slate.
func (db *TestDB) Reset(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if _, err := db.PG.ExecContext(ctx, "TRUNCATE users, emails CASCADE"); err != nil {
		t.Fatalf("testutil: truncate users: %v", err)
	}
	if err := db.Mongo.Collection("users").Drop(ctx); err != nil {
		t.Fatalf("testutil: drop users collection: %v", err)
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

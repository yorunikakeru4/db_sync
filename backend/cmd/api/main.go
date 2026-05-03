package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"backend/internal/kafka"
	"backend/internal/storage"
	httptransport "backend/internal/transport/http"
)

// Config contains runtime configuration for the backend API.
type Config struct {
	// HTTPAddr is the listen address for the HTTP server.
	HTTPAddr string
	// PostgresDSN is the PostgreSQL DSN for bun write-side access.
	PostgresDSN string
	// MongoURI is the MongoDB URI for read-side access.
	MongoURI string
	// MongoDBName is the MongoDB database name for read-side access.
	MongoDBName string
	// KafkaBrokers is the broker list for event publishing.
	KafkaBrokers []string
	// KafkaTopic is the event topic for backend mutations.
	KafkaTopic string
}

func main() {
	cfg := mustLoadConfig()

	bunDB, err := storage.OpenBunDB(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer bunDB.Close()

	mongoClient, mongoDB, err := storage.OpenMongoDB(cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatalf("open mongo: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	producer := kafka.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer producer.Close()

	handler, err := httptransport.NewHandler(httptransport.HandlerParams{
		Producer:  producer,
		Users:     storage.NewUserRepo(bunDB),
		Messages:  storage.NewMessageRepo(bunDB),
		Contacts:  storage.NewContactRepo(bunDB),
		UserViews: storage.NewUserViewRepo(mongoDB),
	})
	if err != nil {
		log.Fatalf("build handler: %v", err)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("backend api listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen http: %v", err)
		}
	}()

	waitForShutdown(server)
}

func mustLoadConfig() Config {
	postgresUser := getenv("POSTGRES_USER", "db_user")
	postgresPassword := getenv("POSTGRES_PASSWORD", "db_password")
	postgresDB := getenv("POSTGRES_DB", "db_name")
	postgresHost := getenv("TEST_POSTGRES_HOST", getenv("POSTGRES_HOST", "localhost"))
	postgresPort := getenv("TEST_POSTGRES_PORT", getenv("POSTGRES_PORT", "5432"))
	mongoHost := getenv("TEST_MONGO_HOST", getenv("MONGO_HOST", "localhost"))
	mongoPort := getenv("TEST_MONGO_PORT", getenv("MONGO_PORT", "27017"))
	mongoUser := getenv("MONGO_INITDB_ROOT_USERNAME", "mongo_user")
	mongoPassword := getenv("MONGO_INITDB_ROOT_PASSWORD", "mongo_password")
	kafkaHost := getenv("KAFKA_HOST", "localhost")
	kafkaPort := getenv("KAFKA_PORT", "9092")

	return Config{
		HTTPAddr:    getenv("BACKEND_HTTP_ADDR", ":8080"),
		PostgresDSN: fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB),
		MongoURI:    fmt.Sprintf("mongodb://%s:%s@%s:%s/?authSource=admin", mongoUser, mongoPassword, mongoHost, mongoPort),
		MongoDBName: getenv("MONGO_DB_NAME", "email_service"),
		KafkaBrokers: splitCSV(
			getenv("KAFKA_BROKERS", fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)),
		),
		KafkaTopic: getenv("KAFKA_TOPIC", "sync_topic"),
	}
}

func waitForShutdown(server *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown http: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// Package app wires together all components and runs the event loop.
package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"db_sync/internal/logutil"
	"db_sync/internal/middleware"
	"db_sync/internal/service"
	"db_sync/internal/storage"
	"db_sync/internal/transport/kafka"
)

// Run initialises all infrastructure connections, assembles the service graph,
// and starts the Kafka consumer loop. It blocks until a SIGTERM or SIGINT is
// received, then shuts down gracefully.
func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL
	pg, err := storage.InitDB()
	if err != nil {
		log.Fatal("Failed to connect PostgreSQL:", err)
	}
	defer pg.Close()

	// MongoDB
	mongoClient, mongoDB, err := storage.InitMongoDB()
	if err != nil {
		log.Fatal("Failed to connect MongoDB:", err)
	}
	defer mongoClient.Disconnect(context.Background())

	// Repositories
	userRepo := storage.NewPostgresUserRepository(pg)

	userViewRepo := storage.NewMongoUserViewRepository(mongoDB)
	contactViewRepo := storage.NewMongoContactViewRepository(mongoDB)
	messageViewRepo := storage.NewMongoMessageViewRepository(mongoDB)
	// Services
	userService := service.NewUserService(userRepo, userViewRepo)
	contactService := service.NewContactService(contactViewRepo)
	messageService := service.NewMessageService(messageViewRepo)

	syncService := service.NewSyncService(
		userService,
		contactService,
		messageService,
	)

	logger := logutil.NewRuntimeLogger(os.Stdout)

	reader := kafka.InitConsumer()
	consumer := middleware.NewLoggingConsumer(kafka.NewKafkaConsumer(reader), logger)
	defer consumer.Close()

	log.Println("Sync service started. Waiting for events...")

	go waitForExit(cancel)

	for {
		event, err := consumer.GetEvent(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Println("Error reading event:", err)
			time.Sleep(time.Second)
			continue
		}

		if err := consumer.DispatchEvent(ctx, event, syncService); err != nil {
			log.Println("Error processing event:", err)
		}
	}
}

// waitForExit blocks until SIGTERM or SIGINT is received, then cancels ctx.
func waitForExit(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c

	log.Println("Shutting down...")
	cancel()
	time.Sleep(500 * time.Millisecond)
}

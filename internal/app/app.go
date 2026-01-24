package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"db_sync/internal/service"
	"db_sync/internal/storage"
	"db_sync/internal/transport/kafka"
)

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
	emailRepo := storage.NewPostgresEmailRepository(pg)
	messageRepo := storage.NewPostgresMessageRepository(pg)

	userViewRepo := storage.NewMongoUserViewRepository(mongoDB)
	emailViewRepo := storage.NewMongoEmailViewRepository(mongoDB)
	messageViewRepo := storage.NewMongoMessageViewRepository(mongoDB)
	// Services
	userService := service.NewUserService(userRepo, userViewRepo)
	emailService := service.NewEmailService(emailRepo, emailViewRepo)
	messageService := service.NewMessageService(messageRepo, messageViewRepo)

	syncService := service.NewSyncService(
		userService,
		emailService,
		messageService,
	)

	reader := kafka.InitConsumer()
	consumer := kafka.NewKafkaConsumer(reader)
	defer consumer.Close()

	log.Println("Sync service started. Waiting for events...")

	go waitForExit(cancel)

	for {
		event, err := consumer.GetEvent(ctx)
		if err != nil {
			log.Println("Error reading event:", err)
			time.Sleep(time.Second)
			continue
		}

		if err := consumer.DispatchEvent(ctx, event, syncService); err != nil {
			log.Println("Error processing event:", err)
		}
	}
}

func waitForExit(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c

	log.Println("Shutting down...")
	cancel()
	time.Sleep(500 * time.Millisecond)
}

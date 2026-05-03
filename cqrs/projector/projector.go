// Package projector exposes a small public entry point for projecting emitted events into MongoDB.
package projector

import (
	"context"
	"encoding/json"

	"db_sync/internal/application/events"
	"db_sync/internal/service"
	"db_sync/internal/storage"
	kafkatransport "db_sync/internal/transport/kafka"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ProjectJSONEvent dispatches a JSON event envelope through the CQRS projector into MongoDB.
func ProjectJSONEvent(ctx context.Context, db *mongo.Database, body []byte) error {
	var event events.Event
	if err := json.Unmarshal(body, &event); err != nil {
		return err
	}

	syncService := service.NewSyncService(
		service.NewUserService(nil, storage.NewMongoUserViewRepository(db)),
		service.NewEmailService(storage.NewMongoEmailViewRepository(db)),
		service.NewMessageService(storage.NewMongoMessageViewRepository(db)),
	)

	consumer := new(kafkatransport.KafkaConsumer)
	return consumer.DispatchEvent(ctx, &event, syncService)
}

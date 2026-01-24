// Package kafka
package kafka

import (
	"context"
	"encoding/json"

	"db_sync/internal/application/events"
	"db_sync/internal/config"
	"db_sync/internal/service"

	"github.com/mitchellh/mapstructure"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(reader *kafka.Reader) *KafkaConsumer {
	return &KafkaConsumer{reader: reader}
}

func InitConsumer() *kafka.Reader {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{config.KafkaConf.Host + ":" + config.KafkaConf.Port},
		Topic:     config.KafkaConf.Topic,
		Partition: 0,
		MaxBytes:  10e6,
	})
	return r
}

func (kc *KafkaConsumer) GetEvent(ctx context.Context) (*events.Event, error) {
	m, err := kc.reader.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}
	var event events.Event
	err = json.Unmarshal(m.Value, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (kc *KafkaConsumer) DispatchEvent(
	ctx context.Context,
	event *events.Event,
	syncService *service.SyncService,
) error {
	switch event.EventType {
	case "user_created":
		var payload events.UserCreatedPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return err
		}
		return syncService.UserService.HandleUserCreated(ctx, payload)

	case "user_deleted":
		var payload events.UserDeletedPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return err
		}
		return syncService.UserService.HandleUserDeleted(ctx, payload)
	case "message_created":
		var payload events.MessageCreatedPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return err
		}
		return syncService.MessageService.HandleMessageAddedToUser(ctx, payload)
	case "message_deleted":
		var payload events.MessageDeletedPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return err
		}
		return syncService.MessageService.HandleMessageDeletedFromUser(ctx, payload)
	case "email_added":
		var payload events.EmailAddedPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return err
		}
		return syncService.EmailService.HandleEmailAddedToUser(ctx, payload)
	case "email_updated":
		var payload events.EmailUpdatedPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return err
		}
		return syncService.EmailService.HandleEmailUpdatedForUser(ctx, payload)
	case "email_removed":
		var payload events.EmailRemovedPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return err
		}
		return syncService.EmailService.HandleEmailRemovedFromUser(ctx, payload)
	default:
		return nil
	}
}

func (kc *KafkaConsumer) Close() error {
	return kc.reader.Close()
}

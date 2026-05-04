// Package storage provides repository implementations for PostgreSQL and MongoDB.
package storage

import (
	"context"
	"errors"

	"db_sync/internal/domain"
	"db_sync/internal/view"

	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ErrMessageViewUserNotFound indicates that the target user view document does not exist.
var ErrMessageViewUserNotFound = errors.New("message view user not found")

// MessageRepository reads messages from the PostgreSQL write model.
type MessageRepository interface {
	// GetMessagesByUserID returns all messages where the user is sender or receiver.
	GetMessagesByUserID(userID int) ([]domain.Message, error)
	// GetMessageByID returns a single message by its primary key.
	GetMessageByID(id int) (*domain.Message, error)
}

// MessageViewRepository updates the MongoDB read model for messages.
type MessageViewRepository interface {
	// AddMessageToUser appends a message view to the user's messages array.
	AddMessageToUser(ctx context.Context, userID int, message view.MessageView) error
	// UpsertMessageToUser replaces an existing embedded message or appends a new one.
	UpsertMessageToUser(ctx context.Context, userID int, message view.MessageView) error
	// RemoveMessageFromUser removes a message view from the user's messages array.
	RemoveMessageFromUser(ctx context.Context, userID int, messageID int) error
}

// PostgresMessageRepository implements MessageRepository using PostgreSQL.
type PostgresMessageRepository struct {
	DB *sqlx.DB
}

// MongoMessageViewRepository implements MessageViewRepository using MongoDB.
type MongoMessageViewRepository struct {
	DB *mongo.Database
}

// NewPostgresMessageRepository creates a new PostgresMessageRepository.
func NewPostgresMessageRepository(db *sqlx.DB) *PostgresMessageRepository {
	return &PostgresMessageRepository{DB: db}
}

// NewMongoMessageViewRepository creates a new MongoMessageViewRepository.
func NewMongoMessageViewRepository(db *mongo.Database) *MongoMessageViewRepository {
	return &MongoMessageViewRepository{DB: db}
}

// GetMessagesByUserID returns all messages where userID is the sender or receiver.
func (r *PostgresMessageRepository) GetMessagesByUserID(userID int) ([]domain.Message, error) {
	var msgs []domain.Message
	err := r.DB.Select(&msgs, `
        SELECT id, external_id, sender_id, receiver_id, subject, text, date_sent, created_at
        FROM messages
        WHERE sender_id=$1 OR receiver_id=$1
        ORDER BY date_sent DESC
    `, userID)
	return msgs, err
}

// GetMessageByID returns the message with the given primary key.
func (r *PostgresMessageRepository) GetMessageByID(id int) (*domain.Message, error) {
	var msg domain.Message
	err := r.DB.Get(&msg, `
        SELECT id, external_id, sender_id, receiver_id, subject, text, date_sent, created_at
        FROM messages
        WHERE id=$1
    `, id)
	return &msg, err
}

// AddMessageToUser appends message to the messages array of the user document.
func (r *MongoMessageViewRepository) AddMessageToUser(
	ctx context.Context,
	userID int,
	message view.MessageView,
) error {
	result, err := r.DB.Collection("users").UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{
			"$push": bson.M{"messages": message},
			"$inc":  bson.M{"num_messages": 1},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.Join(ErrMessageViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

// UpsertMessageToUser replaces an existing embedded message or appends it when absent.
func (r *MongoMessageViewRepository) UpsertMessageToUser(
	ctx context.Context,
	userID int,
	message view.MessageView,
) error {
	collection := r.DB.Collection("users")

	updateExisting, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID, "messages.id": message.ID},
		bson.M{"$set": bson.M{"messages.$": message}},
	)
	if err != nil {
		return err
	}
	if updateExisting.MatchedCount > 0 {
		return nil
	}

	insertNew, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{
			"$push": bson.M{"messages": message},
			"$inc":  bson.M{"num_messages": 1},
		},
	)
	if err != nil {
		return err
	}
	if insertNew.MatchedCount == 0 {
		return errors.Join(ErrMessageViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

// RemoveMessageFromUser removes the message with messageID from the user's messages array.
func (r *MongoMessageViewRepository) RemoveMessageFromUser(
	ctx context.Context,
	userID int,
	messageID int,
) error {
	result, err := r.DB.Collection("users").UpdateOne(
		ctx,
		bson.M{"id": userID, "messages.id": messageID},
		bson.M{
			"$pull": bson.M{"messages": bson.M{"id": messageID}},
			"$inc":  bson.M{"num_messages": -1},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		// Deleting a missing message is idempotent for an existing user view.
		var existing bson.M
		err = r.DB.Collection("users").FindOne(ctx, bson.M{"id": userID}).Decode(&existing)
		if err == nil {
			return nil
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.Join(ErrMessageViewUserNotFound, mongo.ErrNoDocuments)
		}
		return err
	}
	return nil
}

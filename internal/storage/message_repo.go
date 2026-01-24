package storage

import (
	"context"

	"db_sync/internal/domain"
	"db_sync/internal/view"

	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MessageRepository interface {
	GetMessagesByUserID(userID int) ([]domain.Message, error)
	GetMessageByID(id int) (*domain.Message, error)
}
type MessageViewRepository interface {
	AddMessageToUser(ctx context.Context, userID int, message view.MessageView) error
	RemoveMessageFromUser(ctx context.Context, userID int, messageID int) error
}

type PostgresMessageRepository struct {
	DB *sqlx.DB
}
type MongoMessageViewRepository struct {
	DB *mongo.Database
}

func NewPostgresMessageRepository(db *sqlx.DB) *PostgresMessageRepository {
	return &PostgresMessageRepository{DB: db}
}

func NewMongoMessageViewRepository(db *mongo.Database) *MongoMessageViewRepository {
	return &MongoMessageViewRepository{DB: db}
}

func (r *PostgresMessageRepository) GetMessagesByUserID(userID int) ([]domain.Message, error) {
	var msgs []domain.Message
	err := r.DB.Select(&msgs, `
        SELECT id, external_id, sender_id, receivier_id, subject, text, date_sent, created_at
        FROM messages
        WHERE sender_id=$1 OR receivier_id=$1
        ORDER BY date_sent DESC
    `, userID)
	return msgs, err
}

func (r *PostgresMessageRepository) GetMessageByID(id int) (*domain.Message, error) {
	var msg domain.Message
	err := r.DB.Get(&msg, `
        SELECT id, external_id, sender_id, receivier_id, subject, text, date_sent, created_at
        FROM messages
        WHERE id=$1
    `, id)
	return &msg, err
}

func (r *MongoMessageViewRepository) AddMessageToUser(
	ctx context.Context,
	userID int,
	message view.MessageView,
) error {
	_, err := r.DB.Collection("users").UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$push": bson.M{"messages": message}},
	)
	return err
}

func (r *MongoMessageViewRepository) RemoveMessageFromUser(
	ctx context.Context,
	userID int,
	messageID int,
) error {
	_, err := r.DB.Collection("users").UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$pull": bson.M{"messages": bson.M{"id": messageID}}},
	)
	return err
}

package storage

import (
	"context"

	"backend/internal/bunmodel"

	"github.com/uptrace/bun"
)

// MessageRepo persists messages via bun.
type MessageRepo struct {
	db *bun.DB
}

// NewMessageRepo creates a bun-backed message repository.
func NewMessageRepo(db *bun.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

// CreateMessage inserts a message and mutates it with the generated fields.
func (r *MessageRepo) CreateMessage(ctx context.Context, message *bunmodel.Message) error {
	_, err := r.db.NewInsert().Model(message).Returning("*").Exec(ctx)
	return err
}

// UpdateMessage updates a message and mutates it with the persisted fields.
func (r *MessageRepo) UpdateMessage(ctx context.Context, message *bunmodel.Message) error {
	_, err := r.db.NewUpdate().
		Model(message).
		Column("external_id", "sender_id", "receiver_id", "subject", "text", "date_sent").
		WherePK().
		Returning("*").
		Exec(ctx)
	return err
}

// DeleteMessage removes a message by primary key and returns the deleted row.
func (r *MessageRepo) DeleteMessage(ctx context.Context, id int64) (*bunmodel.Message, error) {
	message := new(bunmodel.Message)
	if err := r.db.NewSelect().Model(message).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}
	if _, err := r.db.NewDelete().Model(message).WherePK().Exec(ctx); err != nil {
		return nil, err
	}
	return message, nil
}

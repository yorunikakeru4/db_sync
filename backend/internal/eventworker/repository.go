package eventworker

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// PgRepository implements Repository using bun.
type PgRepository struct {
	db *bun.DB
}

// NewPgRepository creates a new PostgreSQL-backed repository.
func NewPgRepository(db *bun.DB) *PgRepository {
	return &PgRepository{db: db}
}

// FetchUnpublished returns unpublished events ordered by ID.
func (r *PgRepository) FetchUnpublished(ctx context.Context, limit int) ([]DomainEvent, error) {
	var events []DomainEvent
	err := r.db.NewSelect().
		Model(&events).
		Where("published_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Scan(ctx)
	return events, err
}

// MarkPublished sets published_at for the given event IDs.
func (r *PgRepository) MarkPublished(ctx context.Context, ids []int64) error {
	_, err := r.db.NewUpdate().
		Model((*DomainEvent)(nil)).
		Set("published_at = ?", time.Now().UTC()).
		Where("id IN (?)", bun.In(ids)).
		Exec(ctx)
	return err
}

package storage

import (
	"context"
	"errors"

	"backend/internal/bunmodel"

	"github.com/uptrace/bun"
)

// ErrUserNotFound indicates that the requested user does not exist.
var ErrUserNotFound = errors.New("user not found")

// UserRepo persists users via bun.
type UserRepo struct {
	db *bun.DB
}

// NewUserRepo creates a bun-backed user repository.
func NewUserRepo(db *bun.DB) *UserRepo {
	return &UserRepo{db: db}
}

// CreateUser inserts a user and mutates it with the generated fields.
func (r *UserRepo) CreateUser(ctx context.Context, user *bunmodel.User) error {
	_, err := r.db.NewInsert().Model(user).Returning("*").Exec(ctx)
	return err
}

// UpdateUser updates a user and mutates it with the persisted fields.
func (r *UserRepo) UpdateUser(ctx context.Context, user *bunmodel.User) error {
	_, err := r.db.NewUpdate().Model(user).Column("email").WherePK().Returning("*").Exec(ctx)
	return err
}

// DeleteUser removes a user by primary key.
func (r *UserRepo) DeleteUser(ctx context.Context, id int64) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().
			Model((*bunmodel.Message)(nil)).
			Where("sender_id = ? OR receiver_id = ?", id, id).
			Exec(ctx); err != nil {
			return err
		}

		result, err := tx.NewDelete().Model((*bunmodel.User)(nil)).Where("id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrUserNotFound
		}
		return nil
	})
}

// ListUsers returns all users ordered by primary key.
func (r *UserRepo) ListUsers(ctx context.Context) ([]bunmodel.User, error) {
	users := make([]bunmodel.User, 0)
	if err := r.db.NewSelect().Model(&users).Order("id ASC").Scan(ctx); err != nil {
		return nil, err
	}
	return users, nil
}

package storage

import (
	"context"

	"backend/internal/bunmodel"

	"github.com/uptrace/bun"
)

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
	_, err := r.db.NewDelete().Model((*bunmodel.User)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

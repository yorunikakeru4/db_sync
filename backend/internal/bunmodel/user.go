// Package bunmodel contains bun-backed PostgreSQL models for the backend write side.
package bunmodel

import (
	"time"

	"github.com/uptrace/bun"
)

// User represents a user row in the PostgreSQL users table.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	// ID is the primary key.
	ID int64 `bun:"id,pk,autoincrement"`
	// Email is the user's email address.
	Email string `bun:"email,notnull"`
	// CreatedAt is the UTC timestamp when the user was created.
	CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
}

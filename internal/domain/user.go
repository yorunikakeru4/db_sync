// Package domain contains the core write-model entities mapped from PostgreSQL.
package domain

import "time"

// User represents a user row in the PostgreSQL users table.
type User struct {
	// ID is the primary key.
	ID int `db:"id"`
	// Email is the user's email address.
	Email string `db:"email"`
	// CreatedAt is the UTC timestamp when the user was created.
	CreatedAt time.Time `db:"created_at"`
}

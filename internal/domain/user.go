package domain

import "time"

type User struct {
	ID        int       `db:"id"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

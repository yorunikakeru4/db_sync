package domain

import "time"

type Email struct {
	ID        int       `db:"id"`
	Address   string    `db:"email_address"`
	CreatedAt time.Time `db:"created_at"`
}

type UserEmail struct {
	ID         int       `db:"id"`
	UserID     int       `db:"user_id"`
	EmailID    int       `db:"email_id"`
	Importance int       `db:"importance"`
	Category   int       `db:"category"`
	CreatedAt  time.Time `db:"created_at"`
}

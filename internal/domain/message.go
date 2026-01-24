package domain

import "time"

type Message struct {
	ID         int       `db:"id"`
	ExternalID string    `db:"external_id"`
	SenderID   int       `db:"sender_id"`
	ReceiverID int       `db:"receiver_id"`
	Subject    string    `db:"subject"`
	Text       string    `db:"text"`
	DateSent   time.Time `db:"date_sent"`
	CreatedAt  time.Time `db:"created_at"`
}
type MessageFile struct {
	ID        int       `db:"id"`
	MessageID int       `db:"message_id"`
	FilePath  string    `db:"file_path"`
	Type      string    `db:"type"`
	Size      int       `db:"size"`
	CreatedAt time.Time `db:"created_at"`
}
type MessageStatus struct {
	ID        int       `db:"id"`
	MessageID int       `db:"message_id"`
	Status    string    `db:"status"`
	UpdatedAt time.Time `db:"updated_at"`
}

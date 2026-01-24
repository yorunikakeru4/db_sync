package models

import "time"

type Message struct {
	ID         int
	ExternalID string
	SenderID   int
	ReceiverID int
	Subject    string
	Text       string
	DateSent   string
	CreatedAt  time.Time
	Statuses   []Status
	Files      []File
}

type Status struct {
	ID        int
	MessageID int
	Status    string
	UpdatedAt time.Time
}

// Package domain contains the core business logic and domain models of the application.
package view

import "time"

type UserView struct {
	ID                int                    `bson:"id"`
	Email             string                 `bson:"email"`
	NumMessages       int                    `bson:"num_messages"`
	ImportantContacts []ImportantContactView `bson:"important_contacts"`
	Messages          []MessageView          `bson:"messages"`
	CreatedAt         time.Time              `bson:"created_at"`
}
type ImportantContactView struct {
	Email      string `bson:"email"`
	Category   string `bson:"category"`
	Importance int    `bson:"importance"`
}

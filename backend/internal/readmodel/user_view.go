// Package readmodel contains MongoDB-backed CQRS read-side models served by backend GET endpoints.
package readmodel

import "time"

// UserView is the denormalized MongoDB user document.
type UserView struct {
	// ID is the user's primary key.
	ID int64 `bson:"id" json:"id"`
	// Email is the user's email address.
	Email string `bson:"email" json:"email"`
	// NumMessages is the count of embedded messages.
	NumMessages int `bson:"num_messages" json:"num_messages"`
	// ImportantContacts is the list of important contacts.
	ImportantContacts []ImportantContactView `bson:"important_contacts" json:"important_contacts"`
	// Messages is the list of embedded message summaries.
	Messages []MessageView `bson:"messages" json:"messages"`
	// CreatedAt is the UTC timestamp when the user was created.
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// ImportantContactView is the embedded contact sub-document.
type ImportantContactView struct {
	// Email is the contact address.
	Email string `bson:"email" json:"email"`
	// Category is the contact category.
	Category string `bson:"category" json:"category"`
	// Importance is the contact importance.
	Importance int `bson:"importance" json:"importance"`
}

// MessageView is the embedded message sub-document.
type MessageView struct {
	// ID is the message primary key.
	ID int64 `bson:"id" json:"id"`
	// Subject is the message subject.
	Subject string `bson:"subject" json:"subject"`
	// Text is the message body.
	Text string `bson:"text" json:"text"`
	// CreatedAt is the message timestamp.
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

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
	// ContactID is the contact primary key.
	ContactID int64 `bson:"contact_id" json:"contact_id,omitempty"`
	// Value is the contact value.
	Value string `bson:"value" json:"value"`
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

// ContactRow is a flattened contact row derived from user projections.
type ContactRow struct {
	// UserID is the owner user identifier.
	UserID int64 `json:"user_id"`
	// Value is the contact value.
	Value string `json:"value"`
	// Category is the contact category.
	Category string `json:"category"`
	// Importance is the contact importance.
	Importance int `json:"importance"`
}

// MessageRow is a flattened message row derived from user projections.
type MessageRow struct {
	// UserID is the owner user identifier.
	UserID int64 `json:"user_id"`
	// ID is the message identifier.
	ID int64 `json:"id"`
	// Subject is the message subject.
	Subject string `json:"subject"`
	// Text is the message body.
	Text string `json:"text"`
	// CreatedAt is the message timestamp.
	CreatedAt time.Time `json:"created_at"`
}

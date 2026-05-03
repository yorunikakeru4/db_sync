// Package view contains MongoDB read-model (view) document types.
package view

import "time"

// UserView is the denormalised MongoDB document representing a user in the read model.
// It is updated incrementally as domain events arrive from Kafka.
type UserView struct {
	// ID is the user's primary key, used as the MongoDB filter key.
	ID int `bson:"id"`
	// Email is the user's email address.
	Email string `bson:"email"`
	// NumMessages is the total number of messages associated with the user.
	NumMessages int `bson:"num_messages"`
	// ImportantContacts is the list of contacts marked as important.
	ImportantContacts []ImportantContactView `bson:"important_contacts"`
	// Messages is the list of message summaries for this user.
	Messages []MessageView `bson:"messages"`
	// CreatedAt is the UTC timestamp when the user was created.
	CreatedAt time.Time `bson:"created_at"`
}

// ImportantContactView is an embedded sub-document representing a single important contact.
type ImportantContactView struct {
	// Value is the contact value.
	Value string `bson:"value"`
	// Category is the user-assigned category label for this contact.
	Category string `bson:"category"`
	// Importance is the user-assigned importance level for this contact.
	Importance int `bson:"importance"`
}

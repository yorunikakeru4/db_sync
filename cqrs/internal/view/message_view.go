package view

import "time"

// MessageView is the embedded sub-document representing a message summary
// stored inside a UserView document.
type MessageView struct {
	// ID is the message's primary key.
	ID int `bson:"id"`
	// Subject is the message subject line.
	Subject string `bson:"subject"`
	// Text is the message body.
	Text string `bson:"text"`
	// CreatedAt is the UTC timestamp when the message was sent.
	CreatedAt time.Time `bson:"created_at"`
}

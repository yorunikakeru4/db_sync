// Package models defines the data structures used in the application.
package models

type Email struct {
	ID         int
	Address    string
	Category   string
	Importance int
	CreatedAt  string
}

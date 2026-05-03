// Package service contains the business logic for projecting domain events
// from the PostgreSQL write model into the MongoDB read model.
package service

// SyncService aggregates the domain-specific services used by the Kafka consumer.
type SyncService struct {
	// UserService handles user lifecycle events.
	UserService *UserService
	// EmailService handles email/contact events.
	EmailService *EmailService
	// MessageService handles message lifecycle events.
	MessageService *MessageService
}

// NewSyncService creates a SyncService from the provided domain services.
func NewSyncService(
	userService *UserService,
	emailService *EmailService,
	messageService *MessageService,
) *SyncService {
	return &SyncService{
		UserService:    userService,
		EmailService:   emailService,
		MessageService: messageService,
	}
}

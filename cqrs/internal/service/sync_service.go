// Package service contains the business logic for projecting domain events
// from the PostgreSQL write model into the MongoDB read model.
package service

// SyncService aggregates the domain-specific services used by the Kafka consumer.
type SyncService struct {
	// UserService handles user lifecycle events.
	UserService *UserService
	// ContactService handles contact events.
	ContactService *ContactService
	// MessageService handles message lifecycle events.
	MessageService *MessageService
}

// NewSyncService creates a SyncService from the provided domain services.
func NewSyncService(
	userService *UserService,
	contactService *ContactService,
	messageService *MessageService,
) *SyncService {
	return &SyncService{
		UserService:    userService,
		ContactService: contactService,
		MessageService: messageService,
	}
}

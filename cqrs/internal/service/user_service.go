package service

import (
	"context"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/view"
)

// UserService handles user lifecycle events and updates the MongoDB read model.
type UserService struct {
	userRepo     storage.UserRepository
	userViewRepo storage.UserViewRepository
}

// NewUserService creates a new UserService.
func NewUserService(
	userRepo storage.UserRepository,
	userViewRepo storage.UserViewRepository,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		userViewRepo: userViewRepo,
	}
}

// HandleUserCreated projects a user_created event by creating the user view document.
func (us *UserService) HandleUserCreated(
	ctx context.Context,
	event events.UserCreatedPayload,
) error {
	user := view.UserView{
		ID:                event.ID,
		Email:             event.Email,
		NumMessages:       0,
		CreatedAt:         event.CreatedAt,
		ImportantContacts: []view.ImportantContactView{},
		Messages:          []view.MessageView{},
	}
	return us.userViewRepo.CreateUserView(ctx, user)
}

// HandleUserDeleted projects a user_deleted event by removing the user view document.
func (us *UserService) HandleUserDeleted(
	ctx context.Context,
	event events.UserDeletedPayload,
) error {
	return us.userViewRepo.DeleteUserView(ctx, event.ID)
}

// HandleUserUpdated projects a user_updated event by updating the user email in MongoDB.
func (us *UserService) HandleUserUpdated(
	ctx context.Context,
	event events.UserUpdatedPayload,
) error {
	return us.userViewRepo.UpdateUserViewEmail(ctx, event.ID, event.Email)
}

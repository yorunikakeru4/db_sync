package service

import (
	"context"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/view"
)

type UserService struct {
	userRepo     storage.UserRepository
	userViewRepo storage.UserViewRepository
}

func NewUserService(
	userRepo storage.UserRepository,
	userViewRepo storage.UserViewRepository,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		userViewRepo: userViewRepo,
	}
}

func (us *UserService) HandleUserCreated(
	ctx context.Context,
	event events.UserCreatedPayload,
) error {
	user := view.UserView{
		ID:        event.ID,
		Email:     event.Email,
		CreatedAt: event.CreatedAt,
	}
	return us.userViewRepo.CreateUserView(ctx, user)
}

func (us *UserService) HandleUserDeleted(
	ctx context.Context,
	event events.UserDeletedPayload,
) error {
	return us.userViewRepo.DeleteUserView(ctx, event.ID)
}

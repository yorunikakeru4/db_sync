// Package mocks provides testify/mock implementations of storage repository interfaces.
package mocks

import (
	"context"

	"db_sync/internal/models"
	"db_sync/internal/view"

	"github.com/stretchr/testify/mock"
)

// MockUserViewRepository is a mock implementation of storage.UserViewRepository.
type MockUserViewRepository struct {
	mock.Mock
}

// GetUserViewByID implements storage.UserViewRepository.
func (m *MockUserViewRepository) GetUserViewByID(ctx context.Context, id int) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// CreateUserView implements storage.UserViewRepository.
func (m *MockUserViewRepository) CreateUserView(ctx context.Context, userView view.UserView) error {
	args := m.Called(ctx, userView)
	return args.Error(0)
}

// DeleteUserView implements storage.UserViewRepository.
func (m *MockUserViewRepository) DeleteUserView(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// UpdateUserViewEmail implements storage.UserViewRepository.
func (m *MockUserViewRepository) UpdateUserViewEmail(ctx context.Context, id int, email string) error {
	args := m.Called(ctx, id, email)
	return args.Error(0)
}

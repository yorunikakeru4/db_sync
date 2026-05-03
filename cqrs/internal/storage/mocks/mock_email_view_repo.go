package mocks

import (
	"context"

	"db_sync/internal/view"

	"github.com/stretchr/testify/mock"
)

// MockEmailViewRepository is a mock implementation of storage.EmailViewRepository.
type MockEmailViewRepository struct {
	mock.Mock
}

// AddEmailToUser implements storage.EmailViewRepository.
func (m *MockEmailViewRepository) AddEmailToUser(
	ctx context.Context,
	userID int,
	email view.ImportantContactView,
) error {
	args := m.Called(ctx, userID, email)
	return args.Error(0)
}

// UpdateEmailForUser implements storage.EmailViewRepository.
func (m *MockEmailViewRepository) UpdateEmailForUser(
	ctx context.Context,
	userID int,
	oldEmailAddress string,
	email view.ImportantContactView,
) error {
	args := m.Called(ctx, userID, oldEmailAddress, email)
	return args.Error(0)
}

// RemoveEmailFromUser implements storage.EmailViewRepository.
func (m *MockEmailViewRepository) RemoveEmailFromUser(
	ctx context.Context,
	userID int,
	emailAddress string,
) error {
	args := m.Called(ctx, userID, emailAddress)
	return args.Error(0)
}

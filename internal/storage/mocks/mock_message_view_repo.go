package mocks

import (
	"context"

	"db_sync/internal/view"

	"github.com/stretchr/testify/mock"
)

// MockMessageViewRepository is a mock implementation of storage.MessageViewRepository.
type MockMessageViewRepository struct {
	mock.Mock
}

// AddMessageToUser implements storage.MessageViewRepository.
func (m *MockMessageViewRepository) AddMessageToUser(
	ctx context.Context,
	userID int,
	message view.MessageView,
) error {
	args := m.Called(ctx, userID, message)
	return args.Error(0)
}

// RemoveMessageFromUser implements storage.MessageViewRepository.
func (m *MockMessageViewRepository) RemoveMessageFromUser(
	ctx context.Context,
	userID int,
	messageID int,
) error {
	args := m.Called(ctx, userID, messageID)
	return args.Error(0)
}

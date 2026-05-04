package mocks

import (
	"context"

	"db_sync/internal/view"

	"github.com/stretchr/testify/mock"
)

// MockContactViewRepository is a mock implementation of storage.ContactViewRepository.
type MockContactViewRepository struct {
	mock.Mock
}

// AddContactToUser implements storage.ContactViewRepository.
func (m *MockContactViewRepository) AddContactToUser(
	ctx context.Context,
	userID int,
	contact view.ImportantContactView,
) error {
	args := m.Called(ctx, userID, contact)
	return args.Error(0)
}

// UpdateContactForUser implements storage.ContactViewRepository.
func (m *MockContactViewRepository) UpdateContactForUser(
	ctx context.Context,
	userID int,
	contactID int,
	contact view.ImportantContactView,
) error {
	args := m.Called(ctx, userID, contactID, contact)
	return args.Error(0)
}

// RemoveContactFromUser implements storage.ContactViewRepository.
func (m *MockContactViewRepository) RemoveContactFromUser(
	ctx context.Context,
	userID int,
	contactID int,
) error {
	args := m.Called(ctx, userID, contactID)
	return args.Error(0)
}

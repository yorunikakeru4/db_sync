package mocks

import (
	"db_sync/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of storage.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

// GetUserByID implements storage.UserRepository.
func (m *MockUserRepository) GetUserByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

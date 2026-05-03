package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"db_sync/internal/application/events"
	"db_sync/internal/storage/mocks"
	"db_sync/internal/view"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleUserCreated(t *testing.T) {
	createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		payload   events.UserCreatedPayload
		repoErr   error
		wantErr   bool
	}{
		{
			name:    "success",
			payload: events.UserCreatedPayload{ID: 1, Email: "alice@example.com", CreatedAt: createdAt},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:    "repo error propagated",
			payload: events.UserCreatedPayload{ID: 2, Email: "bob@example.com", CreatedAt: createdAt},
			repoErr: errors.New("mongo write failed"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := new(mocks.MockUserRepository)
			viewRepo := new(mocks.MockUserViewRepository)

			expectedView := view.UserView{
				ID:        tc.payload.ID,
				Email:     tc.payload.Email,
				CreatedAt: tc.payload.CreatedAt,
			}
			viewRepo.On("CreateUserView", context.Background(), expectedView).Return(tc.repoErr)

			svc := NewUserService(userRepo, viewRepo)
			err := svc.HandleUserCreated(context.Background(), tc.payload)

			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.repoErr, err)
			} else {
				require.NoError(t, err)
			}
			viewRepo.AssertExpectations(t)
		})
	}
}

func TestHandleUserDeleted(t *testing.T) {
	tests := []struct {
		name    string
		payload events.UserDeletedPayload
		repoErr error
		wantErr bool
	}{
		{
			name:    "success",
			payload: events.UserDeletedPayload{ID: 1},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:    "repo error propagated",
			payload: events.UserDeletedPayload{ID: 99},
			repoErr: errors.New("document not found"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := new(mocks.MockUserRepository)
			viewRepo := new(mocks.MockUserViewRepository)

			viewRepo.On("DeleteUserView", context.Background(), tc.payload.ID).Return(tc.repoErr)

			svc := NewUserService(userRepo, viewRepo)
			err := svc.HandleUserDeleted(context.Background(), tc.payload)

			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.repoErr, err)
			} else {
				require.NoError(t, err)
			}
			viewRepo.AssertExpectations(t)
		})
	}
}

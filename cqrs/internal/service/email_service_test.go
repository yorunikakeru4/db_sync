package service

import (
	"context"
	"errors"
	"testing"

	"db_sync/internal/application/events"
	"db_sync/internal/storage/mocks"
	"db_sync/internal/view"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleEmailAddedToUser(t *testing.T) {
	tests := []struct {
		name    string
		payload events.EmailAddedPayload
		repoErr error
		wantErr bool
	}{
		{
			name: "success",
			payload: events.EmailAddedPayload{
				UserID:     1,
				Address:    "contact@example.com",
				Category:   "work",
				Importance: 3,
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name: "repo error propagated",
			payload: events.EmailAddedPayload{
				UserID:     2,
				Address:    "other@example.com",
				Category:   "personal",
				Importance: 1,
			},
			repoErr: errors.New("update failed"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockEmailViewRepository)
			expected := view.ImportantContactView{
				Email:      tc.payload.Address,
				Category:   tc.payload.Category,
				Importance: tc.payload.Importance,
			}
			repo.On("AddEmailToUser", context.Background(), tc.payload.UserID, expected).Return(tc.repoErr)

			svc := NewEmailService(repo)
			err := svc.HandleEmailAddedToUser(context.Background(), tc.payload)

			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.repoErr, err)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestHandleEmailUpdatedForUser(t *testing.T) {
	tests := []struct {
		name    string
		payload events.EmailUpdatedPayload
		repoErr error
		wantErr bool
	}{
		{
			name: "success — uses NewCategory and NewImportance",
			payload: events.EmailUpdatedPayload{
				UserID:        1,
				Address:       "contact@example.com",
				OldAddress:    "before@example.com",
				OldCategory:   "personal",
				NewCategory:   "work",
				OldImportance: 1,
				NewImportance: 5,
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name: "repo error propagated",
			payload: events.EmailUpdatedPayload{
				UserID:        3,
				Address:       "x@example.com",
				OldAddress:    "old-x@example.com",
				NewCategory:   "vip",
				NewImportance: 10,
			},
			repoErr: errors.New("no matching document"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockEmailViewRepository)
			expected := view.ImportantContactView{
				Email:      tc.payload.Address,
				Category:   tc.payload.NewCategory,
				Importance: tc.payload.NewImportance,
			}
			repo.On("UpdateEmailForUser", context.Background(), tc.payload.UserID, tc.payload.OldAddress, expected).Return(tc.repoErr)

			svc := NewEmailService(repo)
			err := svc.HandleEmailUpdatedForUser(context.Background(), tc.payload)

			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.repoErr, err)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestHandleEmailRemovedFromUser(t *testing.T) {
	tests := []struct {
		name    string
		payload events.EmailRemovedPayload
		repoErr error
		wantErr bool
	}{
		{
			name:    "success",
			payload: events.EmailRemovedPayload{UserID: 1, Address: "old@example.com"},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:    "repo error propagated",
			payload: events.EmailRemovedPayload{UserID: 2, Address: "gone@example.com"},
			repoErr: errors.New("pull failed"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockEmailViewRepository)
			repo.On("RemoveEmailFromUser", context.Background(), tc.payload.UserID, tc.payload.Address).Return(tc.repoErr)

			svc := NewEmailService(repo)
			err := svc.HandleEmailRemovedFromUser(context.Background(), tc.payload)

			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.repoErr, err)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

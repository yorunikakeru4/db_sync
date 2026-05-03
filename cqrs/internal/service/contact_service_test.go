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

func TestHandleContactAddedToUser(t *testing.T) {
	tests := []struct {
		name    string
		payload events.ContactAddedPayload
		repoErr error
		wantErr bool
	}{
		{
			name: "success",
			payload: events.ContactAddedPayload{
				UserID:     1,
				Value:      "contact@example.com",
				Category:   "work",
				Importance: 3,
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name: "repo error propagated",
			payload: events.ContactAddedPayload{
				UserID:     2,
				Value:      "other@example.com",
				Category:   "personal",
				Importance: 1,
			},
			repoErr: errors.New("update failed"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockContactViewRepository)
			expected := view.ImportantContactView{
				Value:      tc.payload.Value,
				Category:   tc.payload.Category,
				Importance: tc.payload.Importance,
			}
			repo.On("AddContactToUser", context.Background(), tc.payload.UserID, expected).Return(tc.repoErr)

			svc := NewContactService(repo)
			err := svc.HandleContactAddedToUser(context.Background(), tc.payload)

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

func TestHandleContactUpdatedForUser(t *testing.T) {
	tests := []struct {
		name    string
		payload events.ContactUpdatedPayload
		repoErr error
		wantErr bool
	}{
		{
			name: "success — uses NewCategory and NewImportance",
			payload: events.ContactUpdatedPayload{
				UserID:        1,
				Value:         "contact@example.com",
				OldValue:      "before@example.com",
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
			payload: events.ContactUpdatedPayload{
				UserID:        3,
				Value:         "x@example.com",
				OldValue:      "old-x@example.com",
				NewCategory:   "vip",
				NewImportance: 10,
			},
			repoErr: errors.New("no matching document"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockContactViewRepository)
			expected := view.ImportantContactView{
				Value:      tc.payload.Value,
				Category:   tc.payload.NewCategory,
				Importance: tc.payload.NewImportance,
			}
			repo.On("UpdateContactForUser", context.Background(), tc.payload.UserID, tc.payload.OldValue, expected).Return(tc.repoErr)

			svc := NewContactService(repo)
			err := svc.HandleContactUpdatedForUser(context.Background(), tc.payload)

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

func TestHandleContactRemovedFromUser(t *testing.T) {
	tests := []struct {
		name    string
		payload events.ContactRemovedPayload
		repoErr error
		wantErr bool
	}{
		{
			name:    "success",
			payload: events.ContactRemovedPayload{UserID: 1, Value: "old@example.com"},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:    "repo error propagated",
			payload: events.ContactRemovedPayload{UserID: 2, Value: "gone@example.com"},
			repoErr: errors.New("pull failed"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockContactViewRepository)
			repo.On("RemoveContactFromUser", context.Background(), tc.payload.UserID, tc.payload.Value).Return(tc.repoErr)

			svc := NewContactService(repo)
			err := svc.HandleContactRemovedFromUser(context.Background(), tc.payload)

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

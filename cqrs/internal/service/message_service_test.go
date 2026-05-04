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

func TestHandleMessageUpsertToUser(t *testing.T) {
	sentAt := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		payload events.MessageCreatedPayload
		repoErr error
		wantErr bool
	}{
		{
			name: "success — Content mapped to Text, DateSent mapped to CreatedAt",
			payload: events.MessageCreatedPayload{
				MessageID: 10,
				UserID:    1,
				Subject:   "Hello",
				Content:   "World",
				DateSent:  sentAt,
			},
			repoErr: nil,
			wantErr: false,
		},
		{
			name: "repo error propagated",
			payload: events.MessageCreatedPayload{
				MessageID: 11,
				UserID:    2,
				Subject:   "Err",
				Content:   "Body",
				DateSent:  sentAt,
			},
			repoErr: errors.New("push failed"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockMessageViewRepository)
			expected := view.MessageView{
				ID:        tc.payload.MessageID,
				Subject:   tc.payload.Subject,
				Text:      tc.payload.Content,
				CreatedAt: tc.payload.DateSent,
			}
			repo.On("UpsertMessageToUser", context.Background(), tc.payload.UserID, expected).Return(tc.repoErr)

			svc := NewMessageService(repo)
			err := svc.HandleMessageUpsertToUser(context.Background(), tc.payload)

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

func TestHandleMessageDeletedFromUser(t *testing.T) {
	tests := []struct {
		name    string
		payload events.MessageDeletedPayload
		repoErr error
		wantErr bool
	}{
		{
			name:    "success",
			payload: events.MessageDeletedPayload{MessageID: 10, UserID: 1},
			repoErr: nil,
			wantErr: false,
		},
		{
			name:    "repo error propagated",
			payload: events.MessageDeletedPayload{MessageID: 99, UserID: 3},
			repoErr: errors.New("message not found"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mocks.MockMessageViewRepository)
			repo.On("RemoveMessageFromUser", context.Background(), tc.payload.UserID, tc.payload.MessageID).Return(tc.repoErr)

			svc := NewMessageService(repo)
			err := svc.HandleMessageDeletedFromUser(context.Background(), tc.payload)

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

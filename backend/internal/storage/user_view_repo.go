package storage

import (
	"context"

	"backend/internal/readmodel"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// UserViewRepo reads CQRS user projections from MongoDB.
type UserViewRepo struct {
	db *mongo.Database
}

// NewUserViewRepo creates a Mongo-backed user view repository.
func NewUserViewRepo(db *mongo.Database) *UserViewRepo {
	return &UserViewRepo{db: db}
}

// GetUserViewByID returns the denormalized MongoDB user document.
func (r *UserViewRepo) GetUserViewByID(ctx context.Context, id int64) (*readmodel.UserView, error) {
	var user readmodel.UserView
	if err := r.db.Collection("users").FindOne(ctx, bson.M{"id": id}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUserViews returns all denormalized user documents from MongoDB.
func (r *UserViewRepo) ListUserViews(ctx context.Context) ([]readmodel.UserView, error) {
	var users []readmodel.UserView
	cursor, err := r.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// ListMessageViews returns flattened message rows from all user projections.
func (r *UserViewRepo) ListMessageViews(ctx context.Context) ([]readmodel.MessageRow, error) {
	users, err := r.ListUserViews(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]readmodel.MessageRow, 0)
	for _, user := range users {
		for _, msg := range user.Messages {
			rows = append(rows, readmodel.MessageRow{
				UserID:    user.ID,
				ID:        msg.ID,
				Subject:   msg.Subject,
				Text:      msg.Text,
				CreatedAt: msg.CreatedAt,
			})
		}
	}
	return rows, nil
}

// ListContactViews returns flattened contact rows from all user projections.
func (r *UserViewRepo) ListContactViews(ctx context.Context) ([]readmodel.ContactRow, error) {
	users, err := r.ListUserViews(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]readmodel.ContactRow, 0)
	for _, user := range users {
		for _, contact := range user.ImportantContacts {
			rows = append(rows, readmodel.ContactRow{
				UserID:     user.ID,
				Value:      contact.Value,
				Category:   contact.Category,
				Importance: contact.Importance,
			})
		}
	}
	return rows, nil
}

// ListContactViewsByUserID returns flattened contact rows for a single user projection.
func (r *UserViewRepo) ListContactViewsByUserID(ctx context.Context, userID int64) ([]readmodel.ContactRow, error) {
	user, err := r.GetUserViewByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows := make([]readmodel.ContactRow, 0, len(user.ImportantContacts))
	for _, contact := range user.ImportantContacts {
		rows = append(rows, readmodel.ContactRow{
			UserID:     user.ID,
			Value:      contact.Value,
			Category:   contact.Category,
			Importance: contact.Importance,
		})
	}
	return rows, nil
}

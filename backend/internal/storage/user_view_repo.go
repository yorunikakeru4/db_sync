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

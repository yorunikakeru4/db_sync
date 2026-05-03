package storage

import (
	"context"

	"db_sync/internal/application/mapper"
	"db_sync/internal/domain"
	"db_sync/internal/models"
	"db_sync/internal/view"

	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// PostgresUserRepository implements UserRepository using PostgreSQL.
type PostgresUserRepository struct {
	DB *sqlx.DB
}

// MongoUserViewRepository implements UserViewRepository using MongoDB.
type MongoUserViewRepository struct {
	DB *mongo.Database
}

// UserRepository reads user data from the PostgreSQL write model.
type UserRepository interface {
	// GetUserByID returns the user with the given primary key.
	GetUserByID(id int) (*models.User, error)
}

// UserViewRepository manages the MongoDB read model for users.
type UserViewRepository interface {
	// GetUserViewByID returns the denormalised user view from MongoDB.
	GetUserViewByID(ctx context.Context, id int) (*models.User, error)
	// CreateUserView inserts or ignores the user view document (idempotent).
	CreateUserView(ctx context.Context, userView view.UserView) error
	// DeleteUserView removes the user view document for the given id.
	DeleteUserView(ctx context.Context, id int) error
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{DB: db}
}

// NewMongoUserViewRepository creates a new MongoUserViewRepository.
func NewMongoUserViewRepository(db *mongo.Database) *MongoUserViewRepository {
	return &MongoUserViewRepository{DB: db}
}

// GetUserByID fetches the user with the given id from PostgreSQL.
func (r *PostgresUserRepository) GetUserByID(id int) (*models.User, error) {
	var user domain.User
	err := r.DB.Get(&user, "SELECT id, email, created_at FROM users WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return &models.User{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// GetUserViewByID fetches the denormalised user view from MongoDB.
func (r *MongoUserViewRepository) GetUserViewByID(ctx context.Context, id int) (*models.User, error) {
	var userView view.UserView
	err := r.DB.Collection("users").
		FindOne(ctx, bson.M{"id": id}).
		Decode(&userView)
	if err != nil {
		return nil, err
	}
	importantEmails := mapper.MapImportantContacts(userView.ImportantContacts)
	return &models.User{
		ID:              userView.ID,
		Email:           userView.Email,
		ImportantEmails: importantEmails,
		CreatedAt:       userView.CreatedAt,
	}, nil
}

// CreateUserView inserts the user view document if one with the same id does not already exist.
// The operation is idempotent: duplicate user_created events are silently ignored.
func (r *MongoUserViewRepository) CreateUserView(ctx context.Context, userView view.UserView) error {
	collection := r.DB.Collection("users")
	opts := options.UpdateOne().SetUpsert(true)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userView.ID},
		bson.M{"$setOnInsert": userView},
		opts,
	)
	return err
}

// DeleteUserView removes the user view document with the given id.
func (r *MongoUserViewRepository) DeleteUserView(ctx context.Context, id int) error {
	collection := r.DB.Collection("users")
	_, err := collection.DeleteOne(ctx, bson.M{"id": id})
	return err
}

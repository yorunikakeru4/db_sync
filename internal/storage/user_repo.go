package storage

import (
	"context"

	"db_sync/internal/application/mapper"
	"db_sync/internal/domain"
	"db_sync/internal/models"
	"db_sync/internal/view"

	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type (
	PostgresUserRepository struct {
		DB *sqlx.DB
	}
	MongoUserViewRepository struct {
		DB *mongo.Database
	}
	UserRepository interface {
		GetUserByID(id int) (*models.User, error)
	}
	UserViewRepository interface {
		GetUserViewByID(id int) (*models.User, error)
		CreateUserView(ctx context.Context, userView view.UserView) error
		DeleteUserView(ctx context.Context, id int) error
	}
)

func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{DB: db}
}

func NewMongoUserViewRepository(db *mongo.Database) *MongoUserViewRepository {
	return &MongoUserViewRepository{DB: db}
}

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

func (r *MongoUserViewRepository) GetUserViewByID(id int) (*models.User, error) {
	var userView view.UserView
	err := r.DB.Collection("users").
		FindOne(nil, map[string]interface{}{"id": id}).
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

func (r *MongoUserViewRepository) CreateUserView(
	ctx context.Context,
	userView view.UserView,
) error {
	collection := r.DB.Collection("users")
	_, err := collection.InsertOne(ctx, userView)
	if err != nil {
		return err
	}
	return nil
}

func (r *MongoUserViewRepository) DeleteUserView(ctx context.Context, id int) error {
	collection := r.DB.Collection("users")
	_, err := collection.DeleteOne(ctx, map[string]interface{}{"id": id})
	if err != nil {
		return err
	}
	return nil
}

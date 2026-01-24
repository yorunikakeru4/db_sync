package storage

import (
	"context"

	"db_sync/internal/domain"
	"db_sync/internal/view"

	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type EmailRepository interface {
	GetEmailsByUserID(userID int) ([]domain.UserEmail, error)
	GetEmailByID(id int) (*domain.Email, error)
}
type EmailViewRepository interface {
	AddEmailToUser(ctx context.Context, userID int, email view.ImportantContactView) error
	UpdateEmailForUser(ctx context.Context, userID int, email view.ImportantContactView) error
	RemoveEmailFromUser(ctx context.Context, userID int, emailAddress string) error
}
type PostgresEmailRepository struct {
	DB *sqlx.DB
}
type MongoEmailViewRepository struct {
	DB *mongo.Database
}

func NewPostgresEmailRepository(db *sqlx.DB) *PostgresEmailRepository {
	return &PostgresEmailRepository{DB: db}
}

func NewMongoEmailViewRepository(db *mongo.Database) *MongoEmailViewRepository {
	return &MongoEmailViewRepository{DB: db}
}

func (r *PostgresEmailRepository) GetEmailByID(id int) (*domain.Email, error) {
	var email domain.Email
	err := r.DB.Get(&email, `
        SELECT id, email_address, created_at
        FROM emails
        WHERE id=$1
    `, id)
	return &email, err
}

func (r *PostgresEmailRepository) GetEmailsByUserID(userID int) ([]domain.UserEmail, error) {
	var items []domain.UserEmail
	err := r.DB.Select(&items, `
        SELECT user_id, email_id, importance, category, created_at
        FROM users_emails
        WHERE user_id=$1
    `, userID)
	return items, err
}

func (r *MongoEmailViewRepository) AddEmailToUser(
	ctx context.Context,
	userID int,
	email view.ImportantContactView,
) error {
	_, err := r.DB.Collection("users").UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$push": bson.M{"important_contacts": email}},
	)
	return err
}

func (r *MongoEmailViewRepository) UpdateEmailForUser(
	ctx context.Context,
	userID int,
	email view.ImportantContactView,
) error {
	collection := r.DB.Collection("users")
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID, "important_contacts.email": email.Email},
		bson.M{
			"$set": bson.M{
				"important_contacts.$.category":   email.Category,
				"important_contacts.$.importance": email.Importance,
			},
		},
	)
	return err
}

func (r *MongoEmailViewRepository) RemoveEmailFromUser(
	ctx context.Context,
	userID int,
	emailAddress string,
) error {
	collection := r.DB.Collection("users")
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$pull": bson.M{"important_contacts": bson.M{"email": emailAddress}}},
	)
	return err
}

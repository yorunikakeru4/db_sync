package storage

import (
	"context"
	"errors"

	"db_sync/internal/domain"
	"db_sync/internal/view"

	"github.com/jmoiron/sqlx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ErrEmailViewUserNotFound indicates that the target user view document does not exist.
var ErrEmailViewUserNotFound = errors.New("email view user not found")

// EmailRepository reads email data from the PostgreSQL write model.
type EmailRepository interface {
	// GetEmailsByUserID returns all user-email associations for the given user.
	GetEmailsByUserID(userID int) ([]domain.UserEmail, error)
	// GetEmailByID returns a single email record by its primary key.
	GetEmailByID(id int) (*domain.Email, error)
}

// EmailViewRepository updates the MongoDB read model for important contacts.
type EmailViewRepository interface {
	// AddEmailToUser appends an important contact to the user's contacts array.
	AddEmailToUser(ctx context.Context, userID int, email view.ImportantContactView) error
	// UpdateEmailForUser updates an existing contact matched by oldEmailAddress.
	UpdateEmailForUser(ctx context.Context, userID int, oldEmailAddress string, email view.ImportantContactView) error
	// RemoveEmailFromUser removes the contact matching emailAddress from the user's contacts array.
	RemoveEmailFromUser(ctx context.Context, userID int, emailAddress string) error
}

// PostgresEmailRepository implements EmailRepository using PostgreSQL.
type PostgresEmailRepository struct {
	DB *sqlx.DB
}

// MongoEmailViewRepository implements EmailViewRepository using MongoDB.
type MongoEmailViewRepository struct {
	DB *mongo.Database
}

// NewPostgresEmailRepository creates a new PostgresEmailRepository.
func NewPostgresEmailRepository(db *sqlx.DB) *PostgresEmailRepository {
	return &PostgresEmailRepository{DB: db}
}

// NewMongoEmailViewRepository creates a new MongoEmailViewRepository.
func NewMongoEmailViewRepository(db *mongo.Database) *MongoEmailViewRepository {
	return &MongoEmailViewRepository{DB: db}
}

// GetEmailByID returns the email record with the given primary key.
func (r *PostgresEmailRepository) GetEmailByID(id int) (*domain.Email, error) {
	var email domain.Email
	err := r.DB.Get(&email, `
        SELECT id, email_address, created_at
        FROM emails
        WHERE id=$1
    `, id)
	return &email, err
}

// GetEmailsByUserID returns all user-email associations for the given user.
func (r *PostgresEmailRepository) GetEmailsByUserID(userID int) ([]domain.UserEmail, error) {
	var items []domain.UserEmail
	err := r.DB.Select(&items, `
        SELECT user_id, email_id, importance, category, created_at
        FROM users_emails
        WHERE user_id=$1
    `, userID)
	return items, err
}

// AddEmailToUser pushes an important contact into the user's important_contacts array.
func (r *MongoEmailViewRepository) AddEmailToUser(
	ctx context.Context,
	userID int,
	email view.ImportantContactView,
) error {
	result, err := r.DB.Collection("users").UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$push": bson.M{"important_contacts": email}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.Join(ErrEmailViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

// UpdateEmailForUser updates category and importance for the contact matching email.Email.
func (r *MongoEmailViewRepository) UpdateEmailForUser(
	ctx context.Context,
	userID int,
	oldEmailAddress string,
	email view.ImportantContactView,
) error {
	collection := r.DB.Collection("users")
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID, "important_contacts.email": oldEmailAddress},
		bson.M{
			"$set": bson.M{
				"important_contacts.$.email":      email.Email,
				"important_contacts.$.category":   email.Category,
				"important_contacts.$.importance": email.Importance,
			},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.Join(ErrEmailViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

// RemoveEmailFromUser pulls the contact matching emailAddress from the user's contacts array.
func (r *MongoEmailViewRepository) RemoveEmailFromUser(
	ctx context.Context,
	userID int,
	emailAddress string,
) error {
	collection := r.DB.Collection("users")
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$pull": bson.M{"important_contacts": bson.M{"email": emailAddress}}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.Join(ErrEmailViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

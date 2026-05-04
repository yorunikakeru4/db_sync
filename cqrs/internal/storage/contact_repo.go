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

// ErrContactViewUserNotFound indicates that the target user view document does not exist.
var ErrContactViewUserNotFound = errors.New("contact view user not found")

// ContactRepository reads contact data from the PostgreSQL write model.
type ContactRepository interface {
	// GetContactsByUserID returns all user-contact associations for the given user.
	GetContactsByUserID(userID int) ([]domain.UserContact, error)
	// GetContactByID returns a single contact record by its primary key.
	GetContactByID(id int) (*domain.Contact, error)
}

// ContactViewRepository updates the MongoDB read model for important contacts.
type ContactViewRepository interface {
	// AddContactToUser appends an important contact to the user's contacts array.
	AddContactToUser(ctx context.Context, userID int, contact view.ImportantContactView) error
	// UpdateContactForUser updates an existing contact matched by contactID.
	UpdateContactForUser(ctx context.Context, userID int, contactID int, contact view.ImportantContactView) error
	// RemoveContactFromUser removes the contact matching contactID from the user's contacts array.
	RemoveContactFromUser(ctx context.Context, userID int, contactID int) error
}

// PostgresContactRepository implements ContactRepository using PostgreSQL.
type PostgresContactRepository struct {
	DB *sqlx.DB
}

// MongoContactViewRepository implements ContactViewRepository using MongoDB.
type MongoContactViewRepository struct {
	DB *mongo.Database
}

// NewPostgresContactRepository creates a new PostgresContactRepository.
func NewPostgresContactRepository(db *sqlx.DB) *PostgresContactRepository {
	return &PostgresContactRepository{DB: db}
}

// NewMongoContactViewRepository creates a new MongoContactViewRepository.
func NewMongoContactViewRepository(db *mongo.Database) *MongoContactViewRepository {
	return &MongoContactViewRepository{DB: db}
}

// GetContactByID returns the contact record with the given primary key.
func (r *PostgresContactRepository) GetContactByID(id int) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.DB.Get(&contact, `
        SELECT id, contact_value, created_at
        FROM contacts
        WHERE id=$1
    `, id)
	return &contact, err
}

// GetContactsByUserID returns all user-contact associations for the given user.
func (r *PostgresContactRepository) GetContactsByUserID(userID int) ([]domain.UserContact, error) {
	var items []domain.UserContact
	err := r.DB.Select(&items, `
        SELECT user_id, contact_id, importance, category, created_at
        FROM users_contacts
        WHERE user_id=$1
    `, userID)
	return items, err
}

// AddContactToUser pushes an important contact into the user's important_contacts array.
func (r *MongoContactViewRepository) AddContactToUser(
	ctx context.Context,
	userID int,
	contact view.ImportantContactView,
) error {
	result, err := r.DB.Collection("users").UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$push": bson.M{"important_contacts": contact}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.Join(ErrContactViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

// UpdateContactForUser updates the contact matching contactID.
func (r *MongoContactViewRepository) UpdateContactForUser(
	ctx context.Context,
	userID int,
	contactID int,
	contact view.ImportantContactView,
) error {
	collection := r.DB.Collection("users")
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID, "important_contacts.contact_id": contactID},
		bson.M{
			"$set": bson.M{
				"important_contacts.$.value":      contact.Value,
				"important_contacts.$.category":   contact.Category,
				"important_contacts.$.importance": contact.Importance,
			},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.Join(ErrContactViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

// RemoveContactFromUser pulls the contact matching contactID from the user's contacts array.
func (r *MongoContactViewRepository) RemoveContactFromUser(
	ctx context.Context,
	userID int,
	contactID int,
) error {
	collection := r.DB.Collection("users")
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"id": userID},
		bson.M{"$pull": bson.M{"important_contacts": bson.M{"contact_id": contactID}}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.Join(ErrContactViewUserNotFound, mongo.ErrNoDocuments)
	}
	return nil
}

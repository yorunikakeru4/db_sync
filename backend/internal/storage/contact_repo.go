package storage

import (
	"context"
	"database/sql"
	"time"

	"backend/internal/bunmodel"

	"github.com/uptrace/bun"
)

// ContactUpdateSnapshot captures the previous association state before an update.
type ContactUpdateSnapshot struct {
	OldValue      string
	OldCategory   int
	OldImportance int
}

// DeletedUserContact captures the state removed by DeleteUserContact.
type DeletedUserContact struct {
	Contact     bunmodel.Contact
	UserContact bunmodel.UserContact
}

// UserContactWithValue is a denormalized contact row for API reads.
type UserContactWithValue struct {
	UserID           int64     `json:"user_id" bun:"user_id"`
	ContactID        int64     `json:"contact_id" bun:"contact_id"`
	Value            string    `json:"value" bun:"value"`
	Importance       int       `json:"importance" bun:"importance"`
	Category         int       `json:"category" bun:"category"`
	CreatedAt        time.Time `json:"created_at" bun:"created_at"`
	ContactCreatedAt time.Time `json:"contact_created_at" bun:"contact_created_at"`
}

// ContactRepo persists contacts and user-contact links via bun.
type ContactRepo struct {
	db *bun.DB
}

// NewContactRepo creates a bun-backed contact repository.
func NewContactRepo(db *bun.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

// AddUserContact creates or reuses a contact and links it to a user.
func (r *ContactRepo) AddUserContact(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if contact.ID != 0 {
			if _, err := tx.NewInsert().Model(contact).On("CONFLICT (id) DO UPDATE").Set("contact_value = EXCLUDED.contact_value").Exec(ctx); err != nil {
				return err
			}
		} else {
			if _, err := tx.NewInsert().Model(contact).On("CONFLICT (contact_value) DO UPDATE").Set("contact_value = EXCLUDED.contact_value").Returning("*").Exec(ctx); err != nil {
				return err
			}
		}
		userContact.ContactID = contact.ID
		if _, err := tx.NewInsert().Model(userContact).
			Column("user_id", "contact_id", "importance", "category").
			On("CONFLICT (user_id, contact_id) DO UPDATE").
			Set("importance = EXCLUDED.importance").
			Set("category = EXCLUDED.category").
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}

		return tx.NewSelect().Model(contact).Where("id = ?", contact.ID).Scan(ctx)
	})
}

// UpdateUserContact updates both the contact row and association metadata.
func (r *ContactRepo) UpdateUserContact(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) (*ContactUpdateSnapshot, error) {
	snapshot := new(ContactUpdateSnapshot)
	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		current := new(bunmodel.UserContact)
		if err := tx.NewSelect().Model(current).Where("user_id = ? AND contact_id = ?", userContact.UserID, userContact.ContactID).Scan(ctx); err != nil {
			return err
		}
		currentContact := new(bunmodel.Contact)
		if err := tx.NewSelect().Model(currentContact).Where("id = ?", userContact.ContactID).Scan(ctx); err != nil {
			return err
		}
		snapshot.OldValue = currentContact.Value
		snapshot.OldCategory = current.Category
		snapshot.OldImportance = current.Importance

		if _, err := tx.NewUpdate().Model(contact).Column("contact_value").WherePK().Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewUpdate().
			Model(userContact).
			Column("importance", "category").
			Where("user_id = ? AND contact_id = ?", userContact.UserID, userContact.ContactID).
			Exec(ctx); err != nil {
			return err
		}
		if err := tx.NewSelect().Model(contact).Where("id = ?", contact.ID).Scan(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(userContact).Where("user_id = ? AND contact_id = ?", userContact.UserID, userContact.ContactID).Scan(ctx)
	})
	return snapshot, err
}

// DeleteUserContact removes the user-contact link and returns the deleted state.
func (r *ContactRepo) DeleteUserContact(ctx context.Context, userID, contactID int64) (*DeletedUserContact, error) {
	deleted := new(DeletedUserContact)
	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().Model(&deleted.Contact).Where("id = ?", contactID).Scan(ctx); err != nil {
			return err
		}
		if err := tx.NewSelect().Model(&deleted.UserContact).Where("user_id = ? AND contact_id = ?", userID, contactID).Scan(ctx); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model(&deleted.UserContact).Where("user_id = ? AND contact_id = ?", userID, contactID).Exec(ctx); err != nil {
			return err
		}
		count, err := tx.NewSelect().Model((*bunmodel.UserContact)(nil)).Where("contact_id = ?", contactID).Count(ctx)
		if err != nil {
			return err
		}
		if count == 0 {
			if _, err := tx.NewDelete().Model(&deleted.Contact).WherePK().Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	return deleted, err
}

// ListUserContacts returns all user-contact links joined with contact values.
func (r *ContactRepo) ListUserContacts(ctx context.Context) ([]UserContactWithValue, error) {
	rows := make([]UserContactWithValue, 0)
	err := r.db.NewSelect().
		TableExpr("users_contacts AS uc").
		Join("JOIN contacts AS c ON c.id = uc.contact_id").
		ColumnExpr("uc.user_id").
		ColumnExpr("uc.contact_id").
		ColumnExpr("c.contact_value AS value").
		ColumnExpr("uc.importance").
		ColumnExpr("uc.category").
		ColumnExpr("uc.created_at").
		ColumnExpr("c.created_at AS contact_created_at").
		OrderExpr("uc.user_id ASC, uc.contact_id ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListUserContactsByUserID returns all contact links for a single user.
func (r *ContactRepo) ListUserContactsByUserID(ctx context.Context, userID int64) ([]UserContactWithValue, error) {
	rows := make([]UserContactWithValue, 0)
	err := r.db.NewSelect().
		TableExpr("users_contacts AS uc").
		Join("JOIN contacts AS c ON c.id = uc.contact_id").
		Where("uc.user_id = ?", userID).
		ColumnExpr("uc.user_id").
		ColumnExpr("uc.contact_id").
		ColumnExpr("c.contact_value AS value").
		ColumnExpr("uc.importance").
		ColumnExpr("uc.category").
		ColumnExpr("uc.created_at").
		ColumnExpr("c.created_at AS contact_created_at").
		OrderExpr("uc.contact_id ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

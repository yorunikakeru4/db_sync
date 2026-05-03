package storage

import (
	"context"
	"database/sql"

	"backend/internal/bunmodel"

	"github.com/uptrace/bun"
)

// EmailUpdateSnapshot captures the previous association state before an update.
type EmailUpdateSnapshot struct {
	OldAddress    string
	OldCategory   int
	OldImportance int
}

// DeletedUserEmail captures the state removed by DeleteUserEmail.
type DeletedUserEmail struct {
	Email     bunmodel.Email
	UserEmail bunmodel.UserEmail
}

// EmailRepo persists emails and user-email links via bun.
type EmailRepo struct {
	db *bun.DB
}

// NewEmailRepo creates a bun-backed email repository.
func NewEmailRepo(db *bun.DB) *EmailRepo {
	return &EmailRepo{db: db}
}

// AddUserEmail creates or reuses an email and links it to a user.
func (r *EmailRepo) AddUserEmail(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if email.ID != 0 {
			if _, err := tx.NewInsert().Model(email).On("CONFLICT (id) DO UPDATE").Set("email_address = EXCLUDED.email_address").Exec(ctx); err != nil {
				return err
			}
		} else {
			if _, err := tx.NewInsert().Model(email).On("CONFLICT (email_address) DO UPDATE").Set("email_address = EXCLUDED.email_address").Returning("*").Exec(ctx); err != nil {
				return err
			}
		}
		userEmail.EmailID = email.ID
		_, err := tx.NewInsert().Model(userEmail).
			Column("user_id", "email_id", "importance", "category").
			On("CONFLICT (user_id, email_id) DO UPDATE").
			Set("importance = EXCLUDED.importance").
			Set("category = EXCLUDED.category").
			Exec(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			"UPDATE users_emails SET importance = ?, category = ? WHERE user_id = ? AND email_id = ?",
			userEmail.Importance, userEmail.Category, userEmail.UserID, userEmail.EmailID,
		); err != nil {
			return err
		}

		if err := tx.NewSelect().Model(email).Where("id = ?", email.ID).Scan(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(userEmail).Where("user_id = ? AND email_id = ?", userEmail.UserID, userEmail.EmailID).Scan(ctx)
	})
}

// UpdateUserEmail updates both the email row and association metadata.
func (r *EmailRepo) UpdateUserEmail(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) (*EmailUpdateSnapshot, error) {
	snapshot := new(EmailUpdateSnapshot)
	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		current := new(bunmodel.UserEmail)
		if err := tx.NewSelect().Model(current).Where("user_id = ? AND email_id = ?", userEmail.UserID, userEmail.EmailID).Scan(ctx); err != nil {
			return err
		}
		currentEmail := new(bunmodel.Email)
		if err := tx.NewSelect().Model(currentEmail).Where("id = ?", userEmail.EmailID).Scan(ctx); err != nil {
			return err
		}
		snapshot.OldAddress = currentEmail.Address
		snapshot.OldCategory = current.Category
		snapshot.OldImportance = current.Importance

		if _, err := tx.NewUpdate().Model(email).Column("email_address").WherePK().Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewUpdate().
			Model(userEmail).
			Column("importance", "category").
			Where("user_id = ? AND email_id = ?", userEmail.UserID, userEmail.EmailID).
			Exec(ctx); err != nil {
			return err
		}
		if err := tx.NewSelect().Model(email).Where("id = ?", email.ID).Scan(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(userEmail).Where("user_id = ? AND email_id = ?", userEmail.UserID, userEmail.EmailID).Scan(ctx)
	})
	return snapshot, err
}

// DeleteUserEmail removes the user-email link and returns the deleted state.
func (r *EmailRepo) DeleteUserEmail(ctx context.Context, userID, emailID int64) (*DeletedUserEmail, error) {
	deleted := new(DeletedUserEmail)
	err := r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().Model(&deleted.Email).Where("id = ?", emailID).Scan(ctx); err != nil {
			return err
		}
		if err := tx.NewSelect().Model(&deleted.UserEmail).Where("user_id = ? AND email_id = ?", userID, emailID).Scan(ctx); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model(&deleted.UserEmail).Where("user_id = ? AND email_id = ?", userID, emailID).Exec(ctx); err != nil {
			return err
		}
		count, err := tx.NewSelect().Model((*bunmodel.UserEmail)(nil)).Where("email_id = ?", emailID).Count(ctx)
		if err != nil {
			return err
		}
		if count == 0 {
			if _, err := tx.NewDelete().Model(&deleted.Email).WherePK().Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	return deleted, err
}

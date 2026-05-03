package dbfixture

import (
	"context"
	"fmt"

	"github.com/brianvoe/gofakeit/v5"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// ---------------------------------------------------------------------------
// UserLoader
// ---------------------------------------------------------------------------

// UserLoader loads user fixtures.
type UserLoader struct {
	pgLoaderBase[seedinput.User, models.User]

	// confirmedEmails controls whether user emails are marked as confirmed.
	confirmedEmails bool
}

// NewUserLoader creates a Loader for users.
func NewUserLoader(f *Fixture, params LoaderParams) *UserLoader {
	return &UserLoader{pgLoaderBase: pgLoaderBase[seedinput.User, models.User]{
		params: params, fixture: f, name: ModelUser,
	}}
}

func (l *UserLoader) Defaults(ctx context.Context, input *seedinput.User, fake bool) error {
	if !l.confirmedEmails {
		input.EmailConfirmed = nil
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = l.fixture.clock.Now()
	}

	if fake {
		if input.Name == nil {
			name := gofakeit.Name()
			input.Name = &name
		}
		if input.Email == nil {
			email := gofakeit.Email()
			input.Email = &email
		}
		if input.Password == nil {
			password := gofakeit.Password(true, true, true, true, true, 12)
			input.Password = &password
		}
	}

	// Generate gravatar from email.
	if input.Avatar == nil && input.Email != nil {
		model := new(models.User)
		model.Email = *input.Email
		gravatar := model.Gravatar()
		input.Avatar = &gravatar
	}
	return nil
}

func (l *UserLoader) PopulateModel(ctx context.Context, model *models.User, input *seedinput.User) ([]string, error) {
	var cols []string
	if input.Name != nil && model.Name != *input.Name {
		model.Name = *input.Name
		cols = append(cols, "name")
	}
	if input.Email != nil && model.Email != *input.Email {
		model.Email = *input.Email
		cols = append(cols, "email")
	}
	if input.EmailConfirmed != nil && model.EmailConfirmed != *input.EmailConfirmed {
		model.EmailConfirmed = *input.EmailConfirmed
		cols = append(cols, "email_confirmed")
	}
	if input.Password != nil {
		// Hash the plaintext password before storing it on the model.
		if err := model.SetPassword(*input.Password); err != nil {
			return nil, err
		}
		cols = append(cols, "password")
	}
	if input.Avatar != nil && model.Avatar != *input.Avatar {
		model.Avatar = *input.Avatar
		cols = append(cols, "avatar")
	}
	if !input.CreatedAt.IsZero() && model.CreatedAt != input.CreatedAt {
		model.CreatedAt = input.CreatedAt
		cols = append(cols, "created_at")
	}
	return cols, nil
}

// ApplyOpt implements OptApplier.
func (l *UserLoader) ApplyOpt(opt any) {
	switch o := opt.(type) {
	case confirmedEmailsOpt:
		l.confirmedEmails = bool(o)
	}
}

// ---------------------------------------------------------------------------
// UserTokenLoader
// ---------------------------------------------------------------------------

// UserTokenLoader loads user token fixtures.
type UserTokenLoader struct {
	pgLoaderBase[seedinput.UserToken, models.UserToken]
}

// NewUserTokenLoader creates a Loader for user tokens.
func NewUserTokenLoader(f *Fixture, params LoaderParams) *UserTokenLoader {
	return &UserTokenLoader{pgLoaderBase: pgLoaderBase[seedinput.UserToken, models.UserToken]{
		params: params, fixture: f, name: ModelUserToken,
	}}
}

func (l *UserTokenLoader) Resolve(ctx context.Context, input *seedinput.UserToken) error {
	user, ok := Get[*models.User](l.fixture, input.UserKey)
	if !ok {
		return fmt.Errorf("user %q not found", input.UserKey)
	}
	input.UserID = user.ID
	return nil
}

func (l *UserTokenLoader) Defaults(ctx context.Context, input *seedinput.UserToken, fake bool) error {
	if fake && input.Token == "" {
		input.Token = gofakeit.UUID()
	}
	return nil
}

func (l *UserTokenLoader) PopulateModel(ctx context.Context, model *models.UserToken, input *seedinput.UserToken) ([]string, error) {
	var cols []string
	if input.UserID != 0 && model.UserID != input.UserID {
		model.UserID = input.UserID
		cols = append(cols, "user_id")
	}
	if input.Token != "" && model.Token != input.Token {
		model.Token = input.Token
		cols = append(cols, "token")
	}
	if !input.CreatedAt.IsZero() && model.CreatedAt != input.CreatedAt {
		model.CreatedAt = input.CreatedAt
		cols = append(cols, "created_at")
	}
	return cols, nil
}

// ---------------------------------------------------------------------------
// OrgUserLoader
// ---------------------------------------------------------------------------

// OrgUserLoader loads org-user relationship fixtures.
type OrgUserLoader struct {
	pgLoaderBase[seedinput.OrgUser, models.OrgUser]
}

// NewOrgUserLoader creates a Loader for org-user relationships.
func NewOrgUserLoader(f *Fixture, params LoaderParams) *OrgUserLoader {
	return &OrgUserLoader{pgLoaderBase: pgLoaderBase[seedinput.OrgUser, models.OrgUser]{
		params: params, fixture: f, name: ModelOrgUser,
	}}
}

func (l *OrgUserLoader) Resolve(ctx context.Context, input *seedinput.OrgUser) error {
	org, ok := Get[*models.Org](l.fixture, input.OrgKey)
	if !ok {
		return fmt.Errorf("org %q not found", input.OrgKey)
	}
	input.OrgID = org.ID

	user, ok := Get[*models.User](l.fixture, input.UserKey)
	if !ok {
		return fmt.Errorf("user %q not found", input.UserKey)
	}
	input.UserID = user.ID
	return nil
}

func (l *OrgUserLoader) PopulateModel(ctx context.Context, model *models.OrgUser, input *seedinput.OrgUser) ([]string, error) {
	var cols []string
	if input.OrgID != 0 && model.OrgID != input.OrgID {
		model.OrgID = input.OrgID
		cols = append(cols, "org_id")
	}
	if input.UserID != 0 && model.UserID != input.UserID {
		model.UserID = input.UserID
		cols = append(cols, "user_id")
	}
	if input.Role != nil && model.Role != *input.Role {
		model.Role = *input.Role
		cols = append(cols, "role")
	}
	return cols, nil
}

func (l *OrgUserLoader) Insert(ctx context.Context, model *models.OrgUser) (bool, error) {
	db := l.DB(ctx)
	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	// Re-select to get auto-generated fields.
	if err := db.NewSelect().Model(model).
		Where("org_id = ?", model.OrgID).
		Where("user_id = ?", model.UserID).
		Scan(ctx); err != nil {
		return false, err
	}
	return n != 0, nil
}

func (l *OrgUserLoader) Update(ctx context.Context, model *models.OrgUser, columns []string) error {
	return nil // OrgUser changes are handled by re-insert
}

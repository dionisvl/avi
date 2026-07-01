package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
	"github.com/dionisvl/avi/api-go/internal/platform/locale"
)

type Repository interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateVerifyCode(ctx context.Context, id uuid.UUID, code string, expiry time.Time) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID) error
	SetResetCode(ctx context.Context, id uuid.UUID, code string, expiry time.Time) error
	ClearResetCode(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
	IncrementTokenVersion(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, in UpdateInput) error
	PrepareAccountDeletion(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddRole(ctx context.Context, id uuid.UUID, role string) error
}

type UpdateInput struct {
	Name        *string
	Preferences *model.UserPreferences
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *model.User) error {
	loc := user.Locale
	if loc == "" {
		loc = locale.Default
	}
	user.Locale = loc

	query := `
		INSERT INTO users (id, email, password_hash, roles, locale, is_email_verified, email_verify_code, email_verify_code_expiry, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var verifyExpiry *time.Time
	if !user.EmailVerifyCodeExpiry.IsZero() {
		verifyExpiry = &user.EmailVerifyCodeExpiry
	}

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Roles.ToStrings(),
		user.Locale,
		user.IsEmailVerified,
		user.EmailVerifyCode,
		verifyExpiry,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, roles, token_version, locale, name,
		       preferences, is_email_verified, email_verify_code, email_verify_code_expiry,
		       reset_code, reset_code_expiry, created_at, updated_at
		FROM users WHERE email = $1
	`

	user := &model.User{}
	var resetCodeExpiry *time.Time
	var emailVerifyCodeExpiry *time.Time
	var emailVerifyCode *string
	var resetCode *string
	var name *string
	var rawRoles []string
	var prefJSON []byte

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&rawRoles,
		&user.TokenVersion,
		&user.Locale,
		&name,
		&prefJSON,
		&user.IsEmailVerified,
		&emailVerifyCode,
		&emailVerifyCodeExpiry,
		&resetCode,
		&resetCodeExpiry,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.Roles = model.UserRolesFromStrings(rawRoles)

	if resetCodeExpiry != nil {
		user.ResetCodeExpiry = *resetCodeExpiry
	}
	if emailVerifyCodeExpiry != nil {
		user.EmailVerifyCodeExpiry = *emailVerifyCodeExpiry
	}
	if emailVerifyCode != nil {
		user.EmailVerifyCode = *emailVerifyCode
	}
	if resetCode != nil {
		user.ResetCode = *resetCode
	}
	if name != nil {
		user.Name = *name
	}
	if len(prefJSON) > 0 {
		if err := json.Unmarshal(prefJSON, &user.Preferences); err != nil {
			return nil, fmt.Errorf("unmarshal user preferences: %w", err)
		}
	}

	return user, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, roles, token_version, locale, name,
		       preferences, is_email_verified, email_verify_code, email_verify_code_expiry,
		       reset_code, reset_code_expiry, created_at, updated_at
		FROM users WHERE id = $1
	`

	user := &model.User{}
	var resetCodeExpiry *time.Time
	var emailVerifyCodeExpiry *time.Time
	var emailVerifyCode *string
	var resetCode *string
	var name *string
	var rawRoles []string
	var prefJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&rawRoles,
		&user.TokenVersion,
		&user.Locale,
		&name,
		&prefJSON,
		&user.IsEmailVerified,
		&emailVerifyCode,
		&emailVerifyCodeExpiry,
		&resetCode,
		&resetCodeExpiry,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.Roles = model.UserRolesFromStrings(rawRoles)

	if resetCodeExpiry != nil {
		user.ResetCodeExpiry = *resetCodeExpiry
	}
	if emailVerifyCodeExpiry != nil {
		user.EmailVerifyCodeExpiry = *emailVerifyCodeExpiry
	}
	if emailVerifyCode != nil {
		user.EmailVerifyCode = *emailVerifyCode
	}
	if resetCode != nil {
		user.ResetCode = *resetCode
	}
	if name != nil {
		user.Name = *name
	}
	if len(prefJSON) > 0 {
		if err := json.Unmarshal(prefJSON, &user.Preferences); err != nil {
			return nil, fmt.Errorf("unmarshal user preferences: %w", err)
		}
	}

	return user, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, in UpdateInput) error {
	setClauses := make([]string, 0, 3)
	args := make([]any, 0, 4)
	argPos := 1

	if in.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *in.Name)
		argPos++
	}

	if in.Preferences != nil {
		prefJSON, err := json.Marshal(in.Preferences)
		if err != nil {
			return err
		}
		// || merges into existing JSONB instead of replacing it — preserves unset keys.
		// Cast to jsonb explicitly: pgx encodes []byte as bytea, causing type mismatch with jsonb operator.
		setClauses = append(setClauses, fmt.Sprintf("preferences = preferences || $%d::jsonb", argPos))
		args = append(args, json.RawMessage(prefJSON))
		argPos++
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "),
		argPos,
	)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *repository) PrepareAccountDeletion(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE items
		SET status = 'archived', updated_at = NOW()
		WHERE seller_id = $1
		  AND status <> 'archived'
	`, id); err != nil {
		return fmt.Errorf("archive user items: %w", err)
	}

	return nil
}

func (r *repository) UpdateVerifyCode(ctx context.Context, id uuid.UUID, code string, expiry time.Time) error {
	query := `UPDATE users SET email_verify_code = $1, email_verify_code_expiry = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.Exec(ctx, query, code, expiry, time.Now(), id)
	return err
}

func (r *repository) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET is_email_verified = true, email_verify_code = NULL, updated_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	return err
}

func (r *repository) SetResetCode(ctx context.Context, id uuid.UUID, code string, expiry time.Time) error {
	query := `UPDATE users SET reset_code = $1, reset_code_expiry = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.Exec(ctx, query, code, expiry, time.Now(), id)
	return err
}

func (r *repository) ClearResetCode(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET reset_code = NULL, reset_code_expiry = NULL, updated_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	return err
}

func (r *repository) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	query := `UPDATE users SET password_hash = $1, token_version = token_version + 1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, hash, time.Now(), id)
	return err
}

func (r *repository) IncrementTokenVersion(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET token_version = token_version + 1, updated_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, time.Now(), id)
	return err
}

func (r *repository) AddRole(ctx context.Context, id uuid.UUID, role string) error {
	query := `UPDATE users SET roles = array_append(roles, $1), updated_at = $2 WHERE id = $3 AND NOT ($1 = ANY(roles))`
	_, err := r.db.Exec(ctx, query, role, time.Now(), id)
	return err
}

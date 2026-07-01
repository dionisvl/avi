package session

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
)

type Repository interface {
	// Create inserts a new refresh session.
	Create(ctx context.Context, jti, userID uuid.UUID, expiresAt time.Time) error
	// GetByJTI returns a session by jti; returns (nil, nil) if not found.
	GetByJTI(ctx context.Context, jti uuid.UUID) (*Session, error)
	// Revoke marks a single session as revoked.
	Revoke(ctx context.Context, jti uuid.UUID) error
	// RevokeAll revokes all sessions for a user (used on reuse detection and logout).
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}

type Session struct {
	JTI       uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
	Revoked   bool
}

// IsExpired returns true if the session has passed its expiry time.
func (s *Session) IsExpired(now time.Time) bool {
	return s.ExpiresAt.Before(now)
}

// IsValid returns true if the session is not revoked and not expired.
func (s *Session) IsValid(now time.Time) bool {
	return !s.Revoked && !s.IsExpired(now)
}

// IsReuseAttempt returns true if the session is revoked — indicates a possible token replay attack.
func (s *Session) IsReuseAttempt() bool {
	return s.Revoked
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, jti, userID uuid.UUID, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_sessions (jti, user_id, expires_at) VALUES ($1, $2, $3)`,
		jti, userID, expiresAt,
	)
	return err
}

func (r *repository) GetByJTI(ctx context.Context, jti uuid.UUID) (*Session, error) {
	s := &Session{}
	err := r.db.QueryRow(ctx,
		`SELECT jti, user_id, created_at, expires_at, revoked FROM refresh_sessions WHERE jti = $1`,
		jti,
	).Scan(&s.JTI, &s.UserID, &s.CreatedAt, &s.ExpiresAt, &s.Revoked)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return s, nil
}

func (r *repository) Revoke(ctx context.Context, jti uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE refresh_sessions SET revoked = true WHERE jti = $1`,
		jti,
	)
	return err
}

func (r *repository) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE refresh_sessions SET revoked = true WHERE user_id = $1 AND revoked = false`,
		userID,
	)
	return err
}

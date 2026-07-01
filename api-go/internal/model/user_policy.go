package model

import (
	"time"

	"github.com/google/uuid"
)

// DefaultRoles returns the roles assigned to a newly registered user.
func DefaultRoles() UserRoles {
	return UserRoles{RoleUser}
}

// NewUserInput holds data for constructing a new User.
type NewUserInput struct {
	ID               uuid.UUID
	Email            string
	PasswordHash     string
	Locale           string
	VerifyCode       string
	EmailVerified    bool
	VerifyCodeExpiry time.Time
	Now              time.Time // injectable for tests; zero → time.Now()
}

// NewUser constructs a User ready for registration.
func NewUser(in NewUserInput) *User {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	return &User{
		ID:                    in.ID,
		Email:                 in.Email,
		PasswordHash:          in.PasswordHash,
		Roles:                 DefaultRoles(),
		TokenVersion:          1,
		Locale:                in.Locale,
		IsEmailVerified:       in.EmailVerified,
		EmailVerifyCode:       in.VerifyCode,
		EmailVerifyCodeExpiry: in.VerifyCodeExpiry,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

// IsEmailVerificationValid returns true if the code matches and has not expired.
func (u *User) IsEmailVerificationValid(code string, now time.Time) bool {
	return !u.IsEmailVerified && u.EmailVerifyCode == code && !u.EmailVerifyCodeExpiry.Before(now)
}

// IsResetCodeValid returns true if the reset code matches and has not expired.
func (u *User) IsResetCodeValid(code string, now time.Time) bool {
	return u.ResetCode == code && !u.ResetCodeExpiry.Before(now)
}

// HasValidTokenVersion returns true if the claims token version matches the user's current version.
func (u *User) HasValidTokenVersion(claimsTokenVersion int) bool {
	return u.TokenVersion == claimsTokenVersion
}

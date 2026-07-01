package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dionisvl/avi/api-go/internal/model"
)

func TestNewUser(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	expiry := now.Add(24 * time.Hour)

	u := model.NewUser(model.NewUserInput{
		ID:               id,
		Email:            "test@example.com",
		PasswordHash:     "hash",
		Locale:           "ru",
		VerifyCode:       "123456",
		EmailVerified:    false,
		VerifyCodeExpiry: expiry,
		Now:              now,
	})

	assert.Equal(t, id, u.ID)
	assert.Equal(t, "test@example.com", u.Email)
	assert.Equal(t, "hash", u.PasswordHash)
	assert.Equal(t, "ru", u.Locale)
	assert.Equal(t, "123456", u.EmailVerifyCode)
	assert.False(t, u.IsEmailVerified)
	assert.Equal(t, 1, u.TokenVersion)
	assert.True(t, u.Roles.HasRole(model.RoleUser))
	assert.Equal(t, now, u.CreatedAt)
}

func TestNewUser_EmailVerified(t *testing.T) {
	u := model.NewUser(model.NewUserInput{ID: uuid.New(), Email: "a@b.com", PasswordHash: "h", Locale: "en", EmailVerified: true})
	assert.True(t, u.IsEmailVerified)
}

func TestDefaultRoles(t *testing.T) {
	roles := model.DefaultRoles()
	assert.True(t, roles.HasRole(model.RoleUser))
	assert.False(t, roles.IsAdmin())
}

func TestUser_IsEmailVerificationValid(t *testing.T) {
	now := time.Now()
	validExpiry := now.Add(time.Hour)
	expiredExpiry := now.Add(-time.Second)

	tests := []struct {
		name string
		user model.User
		code string
		want bool
	}{
		{
			name: "already verified → false",
			user: model.User{IsEmailVerified: true, EmailVerifyCode: "111111", EmailVerifyCodeExpiry: validExpiry},
			code: "111111",
			want: false,
		},
		{
			name: "wrong code → false",
			user: model.User{IsEmailVerified: false, EmailVerifyCode: "111111", EmailVerifyCodeExpiry: validExpiry},
			code: "999999",
			want: false,
		},
		{
			name: "expired → false",
			user: model.User{IsEmailVerified: false, EmailVerifyCode: "111111", EmailVerifyCodeExpiry: expiredExpiry},
			code: "111111",
			want: false,
		},
		{
			name: "valid → true",
			user: model.User{IsEmailVerified: false, EmailVerifyCode: "111111", EmailVerifyCodeExpiry: validExpiry},
			code: "111111",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.IsEmailVerificationValid(tt.code, now))
		})
	}
}

func TestUser_IsResetCodeValid(t *testing.T) {
	now := time.Now()
	validExpiry := now.Add(time.Minute)
	expiredExpiry := now.Add(-time.Second)

	tests := []struct {
		name string
		user model.User
		code string
		want bool
	}{
		{
			name: "wrong code → false",
			user: model.User{ResetCode: "123456", ResetCodeExpiry: validExpiry},
			code: "000000",
			want: false,
		},
		{
			name: "expired → false",
			user: model.User{ResetCode: "123456", ResetCodeExpiry: expiredExpiry},
			code: "123456",
			want: false,
		},
		{
			name: "valid → true",
			user: model.User{ResetCode: "123456", ResetCodeExpiry: validExpiry},
			code: "123456",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.IsResetCodeValid(tt.code, now))
		})
	}
}

func TestUser_HasValidTokenVersion(t *testing.T) {
	u := model.User{TokenVersion: 3}
	require.True(t, u.HasValidTokenVersion(3))
	require.False(t, u.HasValidTokenVersion(2))
	require.False(t, u.HasValidTokenVersion(4))
}

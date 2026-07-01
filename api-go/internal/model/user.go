package model

import (
	"time"

	"github.com/google/uuid"
)

// UserPreferences stores default classifieds filter settings for the current user.
// All fields are optional — null means "no preference" (show everything).
type UserPreferences struct {
	CategoryID *string `json:"category_id,omitempty"` // UUID string
	CityID     *string `json:"city_id,omitempty"`     // UUID string
	Condition  *string `json:"condition,omitempty"`
	PriceMin   *int64  `json:"price_min,omitempty"`
	PriceMax   *int64  `json:"price_max,omitempty"`
	Search     *string `json:"search,omitempty"`
}

type User struct {
	ID                    uuid.UUID
	Email                 string
	PasswordHash          string
	Roles                 UserRoles
	TokenVersion          int
	Locale                string
	Name                  string
	AvatarURL             string
	Preferences           UserPreferences
	IsEmailVerified       bool
	EmailVerifyCode       string
	EmailVerifyCodeExpiry time.Time
	ResetCode             string
	ResetCodeExpiry       time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

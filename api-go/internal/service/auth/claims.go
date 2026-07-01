package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims JWT claims with user info
type Claims struct {
	UserID       uuid.UUID `json:"uid"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	TokenVersion int       `json:"tv"`
	jwt.RegisteredClaims
}

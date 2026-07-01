package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type tokenService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func newTokenService(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *tokenService {
	return &tokenService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

type TokenPairWithJTI struct {
	*TokenPair
	RefreshJTI uuid.UUID
	RefreshExp time.Time
}

func (t *tokenService) GenerateTokenPair(userID uuid.UUID, email string, roles []string, tokenVersion int) (*TokenPairWithJTI, error) {
	now := time.Now()

	// Access token
	accessClaims := &Claims{
		UserID:       userID,
		Email:        email,
		Roles:        roles,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(t.accessSecret)
	if err != nil {
		return nil, err
	}

	// Refresh token with jti
	jti, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	refreshExp := now.Add(t.refreshTTL)
	refreshClaims := &Claims{
		UserID:       userID,
		Email:        email,
		Roles:        roles,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti.String(),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(t.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPairWithJTI{
		TokenPair: &TokenPair{
			AccessToken:  accessTokenStr,
			RefreshToken: refreshTokenStr,
			ExpiresIn:    int64(t.accessTTL.Seconds()),
		},
		RefreshJTI: jti,
		RefreshExp: refreshExp,
	}, nil
}

func (t *tokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return t.accessSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrInvalidType
	}

	return claims, nil
}

func (t *tokenService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return t.refreshSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrInvalidType
	}

	return claims, nil
}

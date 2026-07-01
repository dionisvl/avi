package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dionisvl/avi/api-go/internal/service/auth"
)

func TestGetUserMe_Success(t *testing.T) {
	app := newTestApp(t)

	// Register, verify email, and login user (using helper)
	email := "me@example.com"
	password := "password123"
	accessToken := registerVerifyAndLogin(t, app, email, password)

	// Get user info
	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, email, resp["email"])
	assert.NotEmpty(t, resp["id"])
	roles, ok := resp["roles"].([]any)
	assert.True(t, ok)
	assert.Equal(t, 1, len(roles))
}

func TestGetUserMe_WithoutToken(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUserMe_WithInvalidToken(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUserMe_WithExpiredToken(t *testing.T) {
	app := newTestApp(t)

	expiredClaims := &auth.Claims{
		UserID: uuid.Must(uuid.NewV7()),
		Email:  "expired@example.com",
		Roles:  []string{"ROLE_USER"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredToken, err := token.SignedString([]byte("test-access-secret"))
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestRole_CanCreateItem verifies that any authenticated role can create a
// listing directly (the seller is the caller; no owner step).
func TestRole_CanCreateItem(t *testing.T) {
	roles := map[string]string{
		"user":  "ROLE_USER",
		"admin": "ROLE_ADMIN",
	}

	for name, role := range roles {
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t)
			token := registerWithRole(t, app, "role-"+name+"-item-"+uuid.New().String()+"@example.com", "password123", role)

			body := buildCreateItemBody(t, "Sharik-"+uuid.New().String())
			req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		})
	}
}

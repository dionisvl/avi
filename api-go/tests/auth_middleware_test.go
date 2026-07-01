package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuthRequired_WithValidToken(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "test-"+uuid.New().String()+"@example.com", "password123")

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthRequired_WithoutToken(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthRequired_WithInvalidToken(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer invalid_token_xyz")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthRequired_WithMalformedBearerToken(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authhandler "github.com/dionisvl/avi/api-go/internal/api/auth"
)

func TestLogout_RequiresAuth(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogout_InvalidatesOldAccessAndRefreshTokens(t *testing.T) {
	app := newTestApp(t)
	tokens := registerVerifyAndLoginTokens(t, app, "logout-all@example.com", "password123")

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	refreshBody, err := json.Marshal(authhandler.RefreshRequest{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	authhandler "github.com/dionisvl/avi/api-go/internal/api/auth"
)

func TestRefresh_ValidToken(t *testing.T) {
	app := newTestApp(t)

	// Register and login to get refresh token
	loginResp := registerVerifyAndLoginTokens(t, app, "test-"+uuid.New().String()+"@example.com", "password123")
	refreshToken := loginResp.RefreshToken

	// Refresh
	refreshPayload := authhandler.RefreshRequest{
		RefreshToken: refreshToken,
	}

	body, _ := json.Marshal(refreshPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var refreshResp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&refreshResp)
	require.NoError(t, err)
	require.NotEmpty(t, refreshResp["access_token"])
	require.NotEmpty(t, refreshResp["refresh_token"])
}

func TestRefresh_InvalidToken(t *testing.T) {
	app := newTestApp(t)

	refreshPayload := authhandler.RefreshRequest{
		RefreshToken: "invalid-token",
	}

	body, _ := json.Marshal(refreshPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

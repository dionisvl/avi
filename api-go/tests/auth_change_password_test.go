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

func TestChangePassword_WrongCurrentPassword_401(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "chpwd-wrong@example.com", "password123")

	body, err := json.Marshal(authhandler.ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword123",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestChangePassword_Success_ThenLoginWithOldFails(t *testing.T) {
	app := newTestApp(t)
	email := "chpwd-ok@example.com"
	oldPassword := "password123"
	newPassword := "newpassword456"

	tokens := registerVerifyAndLoginTokens(t, app, email, oldPassword)

	// Change password
	body, err := json.Marshal(authhandler.ChangePasswordRequest{
		CurrentPassword: oldPassword,
		NewPassword:     newPassword,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Login with old password → should fail
	loginBody, err := json.Marshal(authhandler.LoginRequest{Email: email, Password: oldPassword})
	require.NoError(t, err)
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	app.ServeHTTP(loginRec, loginReq)

	assert.Equal(t, http.StatusUnauthorized, loginRec.Code)
}

func TestChangePassword_InvalidatesOldAccessAndRefreshTokens(t *testing.T) {
	app := newTestApp(t)
	email := "chpwd-revoke@example.com"
	oldPassword := "password123"
	newPassword := "newpassword456"

	tokens := registerVerifyAndLoginTokens(t, app, email, oldPassword)

	body, err := json.Marshal(authhandler.ChangePasswordRequest{
		CurrentPassword: oldPassword,
		NewPassword:     newPassword,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func TestChangePassword_Success_ThenLoginWithNewSucceeds(t *testing.T) {
	app := newTestApp(t)
	email := "chpwd-new@example.com"
	oldPassword := "password123"
	newPassword := "newpassword456"

	token := registerVerifyAndLogin(t, app, email, oldPassword)

	// Change password
	body, err := json.Marshal(authhandler.ChangePasswordRequest{
		CurrentPassword: oldPassword,
		NewPassword:     newPassword,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Login with new password → success
	loginBody, err := json.Marshal(authhandler.LoginRequest{Email: email, Password: newPassword})
	require.NoError(t, err)
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	app.ServeHTTP(loginRec, loginReq)

	assert.Equal(t, http.StatusOK, loginRec.Code)

	var resp authhandler.LoginResponse
	require.NoError(t, json.NewDecoder(loginRec.Body).Decode(&resp))
	assert.NotEmpty(t, resp.AccessToken)
}

func TestChangePassword_RequiresAuth(t *testing.T) {
	app := newTestApp(t)

	body, err := json.Marshal(authhandler.ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestChangePassword_WeakNewPassword_400(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "chpwd-weak@example.com", "password123")

	body, err := json.Marshal(authhandler.ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "short",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

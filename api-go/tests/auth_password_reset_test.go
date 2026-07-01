package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authhandler "github.com/dionisvl/avi/api-go/internal/api/auth"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
)

func TestPasswordReset_Request_Success(t *testing.T) {
	app := newTestApp(t)

	// First register a user
	registerPayload := authhandler.RegisterRequest{
		Email:    "reset@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Request password reset
	resetReq := authhandler.ResetPasswordRequestReq{
		Email: "reset@example.com",
	}
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Check your email", resp["message"])
}

func TestPasswordReset_Request_NonExistentEmail(t *testing.T) {
	app := newTestApp(t)

	resetReq := authhandler.ResetPasswordRequestReq{
		Email: "nonexistent@example.com",
	}
	body, _ := json.Marshal(resetReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/reset-password/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should return 200 anyway (don't leak email existence)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPasswordReset_InvalidEmail(t *testing.T) {
	app := newTestApp(t)

	resetReq := authhandler.ResetPasswordRequestReq{
		Email: "not-an-email",
	}
	body, _ := json.Marshal(resetReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/reset-password/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPasswordReset_Confirm_InvalidCode(t *testing.T) {
	app := newTestApp(t)

	// Register user first
	registerPayload := authhandler.RegisterRequest{
		Email:    "confirm@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Try to confirm with wrong code
	confirmReq := authhandler.ResetPasswordConfirmReq{
		Email: "confirm@example.com",
		Code:  "wrongcode",
	}
	body, _ = json.Marshal(confirmReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPasswordReset_Confirm_Success(t *testing.T) {
	app := newTestApp(t)

	registerPayload := authhandler.RegisterRequest{
		Email:    "confirm-success@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	resetReq := authhandler.ResetPasswordRequestReq{
		Email: "confirm-success@example.com",
	}
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	code := extractResetCode(t, app, "confirm-success@example.com")

	confirmReq := authhandler.ResetPasswordConfirmReq{
		Email: "confirm-success@example.com",
		Code:  code,
	}
	body, _ = json.Marshal(confirmReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Code confirmed", resp["message"])
}

func TestPasswordReset_Set_InvalidCode(t *testing.T) {
	app := newTestApp(t)

	// Register user first
	registerPayload := authhandler.RegisterRequest{
		Email:    "set@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Try to set new password with wrong code
	setReq := authhandler.ResetPasswordSetReq{
		Email:       "set@example.com",
		Code:        "wrongcode",
		NewPassword: "newpassword123",
	}
	body, _ = json.Marshal(setReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/set", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPasswordReset_Set_ShortPassword(t *testing.T) {
	app := newTestApp(t)

	// Register user first
	registerPayload := authhandler.RegisterRequest{
		Email:    "short@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Try to set password that's too short
	setReq := authhandler.ResetPasswordSetReq{
		Email:       "short@example.com",
		Code:        "anycode",
		NewPassword: "short",
	}
	body, _ = json.Marshal(setReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/set", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPasswordReset_Set_Success(t *testing.T) {
	app := newTestApp(t)

	email := "set-success-" + uuid.New().String() + "@example.com"
	oldPassword := "password123"
	newPassword := "newpassword123"

	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: oldPassword,
	}
	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Verify email before requesting password reset
	verifyCode := getVerificationCode(t, app, email)
	verifyPayload := authhandler.VerifyEmailRequest{
		Email: email,
		Code:  verifyCode,
	}
	body, _ = json.Marshal(verifyPayload)
	req = httptest.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	resetReq := authhandler.ResetPasswordRequestReq{
		Email: email,
	}
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	code := extractResetCode(t, app, email)

	setReq := authhandler.ResetPasswordSetReq{
		Email:       email,
		Code:        code,
		NewPassword: newPassword,
	}
	body, _ = json.Marshal(setReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/set", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Password updated", resp["message"])

	loginOldReq := authhandler.LoginRequest{
		Email:    email,
		Password: oldPassword,
	}
	body, _ = json.Marshal(loginOldReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	loginNewReq := authhandler.LoginRequest{
		Email:    email,
		Password: newPassword,
	}
	body, _ = json.Marshal(loginNewReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	apimiddleware.ResetLimiters()

	confirmReq := authhandler.ResetPasswordConfirmReq{
		Email: email,
		Code:  code,
	}
	body, _ = json.Marshal(confirmReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/reset-password/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

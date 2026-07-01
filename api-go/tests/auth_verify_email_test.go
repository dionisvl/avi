package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	authhandler "github.com/dionisvl/avi/api-go/internal/api/auth"
)

func TestVerifyEmail_Success(t *testing.T) {
	app := newTestApp(t)

	email := "test-" + uuid.New().String() + "@example.com"
	// Register
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Get verification code from DB
	code := getVerificationCode(t, app, email)

	// Verify email
	verifyPayload := authhandler.VerifyEmailRequest{
		Email: email,
		Code:  code,
	}
	body, err = json.Marshal(verifyPayload)
	require.NoError(t, err)
	req = httptest.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Verify user is marked as verified
	var verified bool
	err = app.tx.QueryRow(context.Background(), `SELECT is_email_verified FROM users WHERE email = $1`, email).Scan(&verified)
	require.NoError(t, err)
	require.True(t, verified)
}

func TestVerifyEmail_InvalidCode(t *testing.T) {
	app := newTestApp(t)

	email := "test-" + uuid.New().String() + "@example.com"
	// Register
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Try with wrong code
	verifyPayload := authhandler.VerifyEmailRequest{
		Email: email,
		Code:  "000000",
	}
	body, err = json.Marshal(verifyPayload)
	require.NoError(t, err)
	req = httptest.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.NotEqual(t, http.StatusOK, rec.Code)
}

func TestLogin_BlocksUnverifiedUser(t *testing.T) {
	app := newTestApp(t)

	email := "unverified-" + uuid.New().String() + "@example.com"
	// Register (email not verified)
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Try to login without verification
	loginPayload := authhandler.LoginRequest{
		Email:    email,
		Password: "password123",
	}
	body, err = json.Marshal(loginPayload)
	require.NoError(t, err)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should fail
	require.NotEqual(t, http.StatusOK, rec.Code)
}

func TestLogin_AllowsVerifiedUser(t *testing.T) {
	app := newTestApp(t)

	// Register and verify
	registerVerifyAndLogin(t, app, "test-"+uuid.New().String()+"@example.com", "password123")
}

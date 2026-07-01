package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/dionisvl/avi/api-go/internal/api"
	authhandler "github.com/dionisvl/avi/api-go/internal/api/auth"
)

func TestLogin_Success(t *testing.T) {
	app := newTestApp(t)

	email := "test-" + uuid.New().String() + "@example.com"
	password := "password123"

	// Register and verify user first so this test exercises the login endpoint explicitly.
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: password,
	}

	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	verifyPayload := authhandler.VerifyEmailRequest{
		Email: email,
		Code:  getVerificationCode(t, app, email),
	}

	body, err = json.Marshal(verifyPayload)
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	loginPayload := authhandler.LoginRequest{
		Email:    email,
		Password: password,
	}

	body, err = json.Marshal(loginPayload)
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp authhandler.LoginResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.RefreshToken)
	require.Positive(t, resp.ExpiresIn)
}

func TestLogin_WrongPassword(t *testing.T) {
	app := newTestApp(t)

	email := "test-" + uuid.New().String() + "@example.com"
	// Register
	regPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}

	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Login with wrong password (email not verified yet, but check password validation)
	loginPayload := authhandler.LoginRequest{
		Email:    email,
		Password: "wrongpassword",
	}

	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var resp api.ProblemDetails
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, http.StatusUnauthorized, resp.Status)
	require.Equal(t, "Unauthorized", resp.Title)
	require.Equal(t, "Invalid email or password", resp.Detail)
}

func TestLogin_UnknownEmail(t *testing.T) {
	app := newTestApp(t)

	loginPayload := authhandler.LoginRequest{
		Email:    "unknown@example.com",
		Password: "password123",
	}

	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var resp api.ProblemDetails
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, http.StatusUnauthorized, resp.Status)
	require.Equal(t, "Unauthorized", resp.Title)
	require.Equal(t, "Invalid email or password", resp.Detail)
}

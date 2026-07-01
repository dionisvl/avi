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
	authservice "github.com/dionisvl/avi/api-go/internal/service/auth"
)

func postResendVerification(t *testing.T, app *testApp, email, locale string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(authhandler.ResendVerificationRequest{Email: email, Locale: locale})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/resend-verification", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func TestResendVerification_Success(t *testing.T) {
	app := newTestApp(t)
	authservice.ResetAllAttempts()
	authservice.ResetResendCooldowns()

	email := "resend-" + uuid.New().String() + "@example.com"
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "P@ssword1!",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	oldCode := getVerificationCode(t, app, email)

	rec = postResendVerification(t, app, email, "en")
	require.Equal(t, http.StatusOK, rec.Code)

	newCode := getVerificationCode(t, app, email)
	require.NotEqual(t, oldCode, newCode, "verification code should be rotated after resend")
}

func TestResendVerification_NonexistentEmail_Returns200(t *testing.T) {
	app := newTestApp(t)
	authservice.ResetAllAttempts()
	authservice.ResetResendCooldowns()

	rec := postResendVerification(t, app, "ghost-"+uuid.New().String()+"@example.com", "")
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestResendVerification_AlreadyVerifiedEmail_Returns200(t *testing.T) {
	app := newTestApp(t)
	authservice.ResetAllAttempts()
	authservice.ResetResendCooldowns()

	email := "verified-" + uuid.New().String() + "@example.com"
	registerVerifyAndLogin(t, app, email, "P@ssword1!")

	rec := postResendVerification(t, app, email, "")
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestResendVerification_CooldownReturns429(t *testing.T) {
	app := newTestApp(t)
	authservice.ResetAllAttempts()
	authservice.ResetResendCooldowns()

	email := "cooldown-" + uuid.New().String() + "@example.com"
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "P@ssword1!",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = postResendVerification(t, app, email, "ru")
	require.Equal(t, http.StatusOK, rec.Code)

	rec = postResendVerification(t, app, email, "ru")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRegister_ExistingUnverifiedEmail_ReturnsUserExistsUnverified(t *testing.T) {
	app := newTestApp(t)

	email := "dup-unverified-" + uuid.New().String() + "@example.com"
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "P@ssword1!",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Register again with same unverified email
	body, err = json.Marshal(registerPayload)
	require.NoError(t, err)
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, "USER_EXISTS_UNVERIFIED", resp["code"])
}

func TestRegister_ExistingVerifiedEmail_ReturnsUserAlreadyExists(t *testing.T) {
	app := newTestApp(t)

	email := "dup-verified-" + uuid.New().String() + "@example.com"
	registerVerifyAndLogin(t, app, email, "P@ssword1!")

	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "P@ssword1!",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Equal(t, "USER_ALREADY_EXISTS", resp["code"])
}

func TestResendVerification_NewCodeWorksForVerification(t *testing.T) {
	app := newTestApp(t)
	authservice.ResetAllAttempts()
	authservice.ResetResendCooldowns()

	email := "resend-verify-" + uuid.New().String() + "@example.com"
	registerPayload := authhandler.RegisterRequest{
		Email:    email,
		Password: "P@ssword1!",
	}
	body, err := json.Marshal(registerPayload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = postResendVerification(t, app, email, "ru")
	require.Equal(t, http.StatusOK, rec.Code)

	// Use the new code to verify
	newCode := getVerificationCode(t, app, email)
	verifyPayload := authhandler.VerifyEmailRequest{Email: email, Code: newCode}
	body, err = json.Marshal(verifyPayload)
	require.NoError(t, err)
	req = httptest.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var verified bool
	err = app.tx.QueryRow(context.Background(), `SELECT is_email_verified FROM users WHERE email = $1`, email).Scan(&verified)
	require.NoError(t, err)
	require.True(t, verified)
}

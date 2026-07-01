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

func TestRegister_Success(t *testing.T) {
	app := newTestApp(t)

	email := "test-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp["id"])
	require.Equal(t, email, resp["email"])
}

func TestRegister_EnglishLocale_FromHeader(t *testing.T) {
	app := newTestApp(t)

	email := "test-en-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var locale string
	err = app.tx.QueryRow(req.Context(), `SELECT locale FROM users WHERE email = $1`, email).Scan(&locale)
	require.NoError(t, err)
	require.Equal(t, "en", locale)

	emails := app.emailSender.all()
	require.NotEmpty(t, emails)
	last := emails[len(emails)-1]
	require.Equal(t, "en", last.Locale)
	require.Equal(t, "Verify your email — avi", last.Subject)
	require.Contains(t, last.Body, "Enter this code to verify your email")
}

func TestRegister_EnglishLocale_FromHeaderWithQValue(t *testing.T) {
	app := newTestApp(t)

	email := "test-en-q-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en;q=0.9,ru;q=0.8")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var locale string
	err = app.tx.QueryRow(req.Context(), `SELECT locale FROM users WHERE email = $1`, email).Scan(&locale)
	require.NoError(t, err)
	require.Equal(t, "en", locale)
}

func TestRegister_DefaultLocale_IsEnglish(t *testing.T) {
	app := newTestApp(t)

	email := "test-default-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var locale string
	err = app.tx.QueryRow(req.Context(), `SELECT locale FROM users WHERE email = $1`, email).Scan(&locale)
	require.NoError(t, err)
	require.Equal(t, "en", locale)

	emails := app.emailSender.all()
	require.NotEmpty(t, emails)
	last := emails[len(emails)-1]
	require.Equal(t, "en", last.Locale)
	require.Equal(t, "Verify your email — avi", last.Subject)
}

func TestRegister_RussianLocale_FromHeader(t *testing.T) {
	app := newTestApp(t)

	email := "test-ru-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var locale string
	err = app.tx.QueryRow(req.Context(), `SELECT locale FROM users WHERE email = $1`, email).Scan(&locale)
	require.NoError(t, err)
	require.Equal(t, "ru", locale)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	app := newTestApp(t)

	email := "test-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:    email,
		Password: "password123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	body, _ = json.Marshal(payload)
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestRegister_InvalidEmail(t *testing.T) {
	app := newTestApp(t)

	payload := authhandler.RegisterRequest{
		Email:    "notanemail",
		Password: "password123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_ShortPassword(t *testing.T) {
	app := newTestApp(t)

	email := "test-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:    email,
		Password: "short",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_AdminCanSkipEmailVerification(t *testing.T) {
	app := newTestApp(t)

	adminToken := registerVerifyAndLoginAsAdmin(t, app, "admin-skip-"+uuid.New().String()+"@example.com", "password123")

	email := "verified-user-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:         email,
		Password:      "password123",
		EmailVerified: true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// should be able to login immediately without verify-email step
	loginBody, _ := json.Marshal(authhandler.LoginRequest{Email: email, Password: "password123"})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRegister_EmailVerifiedWithInvalidTokenReturns401(t *testing.T) {
	app := newTestApp(t)

	email := "verified-invalid-token-" + uuid.New().String() + "@example.com"
	payload := authhandler.RegisterRequest{
		Email:         email,
		Password:      "password123",
		EmailVerified: true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRegister_NonAdminEmailVerifiedFlagReturns403(t *testing.T) {
	app := newTestApp(t)

	userToken := registerVerifyAndLogin(t, app, "non-admin-caller-"+uuid.New().String()+"@example.com", "password123")

	payload := authhandler.RegisterRequest{
		Email:         "user-noskip-" + uuid.New().String() + "@example.com",
		Password:      "password123",
		EmailVerified: true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

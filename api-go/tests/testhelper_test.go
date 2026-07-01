package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	authhandler "github.com/dionisvl/avi/api-go/internal/api/auth"
	categoriesapi "github.com/dionisvl/avi/api-go/internal/api/categories"
	chatapi "github.com/dionisvl/avi/api-go/internal/api/chat"
	cityapi "github.com/dionisvl/avi/api-go/internal/api/city"
	contactapi "github.com/dionisvl/avi/api-go/internal/api/contact"
	favoriteapi "github.com/dionisvl/avi/api-go/internal/api/favorite"
	apihealth "github.com/dionisvl/avi/api-go/internal/api/health"
	itemapi "github.com/dionisvl/avi/api-go/internal/api/item"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	paymentapi "github.com/dionisvl/avi/api-go/internal/api/payment"
	uploadapi "github.com/dionisvl/avi/api-go/internal/api/upload"
	apiuser "github.com/dionisvl/avi/api-go/internal/api/user"
	"github.com/dionisvl/avi/api-go/internal/config"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/payment/provider"
	categoryquery "github.com/dionisvl/avi/api-go/internal/query/category"
	chatquery "github.com/dionisvl/avi/api-go/internal/query/chatview"
	favoritequery "github.com/dionisvl/avi/api-go/internal/query/favoriteview"
	itemquery "github.com/dionisvl/avi/api-go/internal/query/item"
	categoryrepo "github.com/dionisvl/avi/api-go/internal/repository/category"
	chatrepo "github.com/dionisvl/avi/api-go/internal/repository/chat"
	cityrepo "github.com/dionisvl/avi/api-go/internal/repository/city"
	favrepo "github.com/dionisvl/avi/api-go/internal/repository/favorite"
	itemrepo "github.com/dionisvl/avi/api-go/internal/repository/item"
	mediarepo "github.com/dionisvl/avi/api-go/internal/repository/media"
	paymentrepo "github.com/dionisvl/avi/api-go/internal/repository/payment"
	sessionrepo "github.com/dionisvl/avi/api-go/internal/repository/session"
	userrepo "github.com/dionisvl/avi/api-go/internal/repository/user"
	authservice "github.com/dionisvl/avi/api-go/internal/service/auth"
	chatservice "github.com/dionisvl/avi/api-go/internal/service/chat"
	contactservice "github.com/dionisvl/avi/api-go/internal/service/contact"
	favservice "github.com/dionisvl/avi/api-go/internal/service/favorite"
	itemservice "github.com/dionisvl/avi/api-go/internal/service/item"
	mediaservice "github.com/dionisvl/avi/api-go/internal/service/media"
	paymentservice "github.com/dionisvl/avi/api-go/internal/service/payment"
	userservice "github.com/dionisvl/avi/api-go/internal/service/user"
)

const testS3BaseURL = "https://test-storage.local"

type testApp struct {
	http.Handler
	tx          pgx.Tx
	emailSender *fakeEmailSender
	storage     *testObjectStorage
}

type testObjectStorage struct {
	mu          sync.Mutex
	deletedKeys []string
}

type sentEmail struct {
	Locale  string
	To      string
	Subject string
	Body    string
}

type fakeEmailSender struct {
	mu     sync.Mutex
	emails []sentEmail
}

func (s *fakeEmailSender) record(locale, to, subject, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emails = append(s.emails, sentEmail{
		Locale:  locale,
		To:      to,
		Subject: subject,
		Body:    body,
	})
	return nil
}

func (s *fakeEmailSender) SendVerificationCode(_ context.Context, locale, to, code string) error {
	if locale == "en" {
		return s.record(locale, to, "Verify your email — avi", "Enter this code to verify your email: "+code)
	}
	return s.record(locale, to, "Подтверждение email — avi", code)
}

func (s *fakeEmailSender) SendPasswordResetCode(_ context.Context, locale, to, code string) error {
	if locale == "en" {
		return s.record(locale, to, "Password reset — avi", "Your password reset code: "+code)
	}
	return s.record(locale, to, "Сброс пароля — avi", code)
}

func (s *fakeEmailSender) SendContactMessage(_ context.Context, locale, to, senderName, senderEmail, subject, message string) error {
	body := senderName + "\n" + senderEmail + "\n" + subject + "\n" + message
	if locale == "en" {
		return s.record(locale, to, "Contact form message — avi: "+subject, body)
	}
	return s.record(locale, to, "Сообщение обратной связи — avi: "+subject, body)
}

func (s *fakeEmailSender) all() []sentEmail {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]sentEmail, len(s.emails))
	copy(out, s.emails)
	return out
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (*testObjectStorage) Upload(_ context.Context, key, _ string, body io.Reader, size int64) (string, error) {
	if size > 0 {
		if _, err := io.Copy(io.Discard, body); err != nil {
			return "", err
		}
	}
	return key, nil
}

func (s *testObjectStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletedKeys = append(s.deletedKeys, key)
	return nil
}

func (s *testObjectStorage) deletedObjectKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.deletedKeys))
	copy(out, s.deletedKeys)
	return out
}

// newTestApp creates a fresh test app with a transaction for DB isolation.
// The transaction is automatically rolled back after the test via t.Cleanup.
// Wiring mirrors internal/app/di.go for the classifieds marketplace.
func newTestApp(t *testing.T) *testApp {
	// Reset rate limiters for clean test state
	apimiddleware.ResetLimiters()

	ctx := context.Background()

	// Begin transaction from shared test pool
	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)

	// Register cleanup to rollback transaction after test
	t.Cleanup(func() {
		_ = tx.Rollback(ctx)
	})

	cfg := &config.Config{
		App: config.AppConfig{
			Env:  "test",
			Port: ":8080",
		},
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    7 * 24 * time.Hour,
		},
		SMTP: config.SMTPConfig{
			Host:           getEnvOrDefault("SMTP_HOST", "mailpit"),
			Port:           1025,
			From:           getEnvOrDefault("SMTP_FROM", "test@avi.app"),
			ContactTo:      "contact-test@avi.app",
			FrontendDomain: "http://localhost:3000",
			User:           "",
			Password:       "",
		},
		Auth: config.AuthConfig{
			ContactRateLimitRPS:   0.01,
			ContactRateLimitBurst: 2,
			ResendCooldown:        60 * time.Second,
		},
		Payments: config.PaymentConfig{
			Currency:                  "RUB",
			PromoteListingAmountMinor: 10000,
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Suppress logs in tests
	}))

	// Repositories (all bound to the test transaction)
	userRepo := userrepo.New(tx)
	itemRepo := itemrepo.New(tx)
	mediaRepo := mediarepo.New(tx)
	favRepo := favrepo.New(tx)
	categoryRepo := categoryrepo.New(tx)
	cityRepo := cityrepo.New(tx)
	sessRepo := sessionrepo.New(tx)
	paymentRepo := paymentrepo.New(tx)

	emailSender := &fakeEmailSender{}
	storage := &testObjectStorage{}

	// Services
	authSvc := authservice.New(userRepo, sessRepo, emailSender, cfg, logger)
	mediaSvc := mediaservice.New(mediaRepo, storage, "test-bucket", testS3BaseURL, logger)
	userSvc := userservice.New(userRepo, mediaSvc, tx, logger)
	itemSvc := itemservice.New(itemRepo, categoryRepo, mediaRepo, tx, logger)
	favSvc := favservice.New(favRepo, logger)
	contactSvc := contactservice.New(emailSender, cfg.SMTP.ContactTo, logger)
	paymentSvc := paymentservice.New(paymentRepo, &noopPaymentProvider{}, itemSvc, userRepo, cfg.Payments.PromoteListingAmountMinor, cfg.Payments.Currency, logger)

	// Query services (CQRS reads)
	itemQuery := itemquery.New(itemRepo, favSvc, testS3BaseURL)
	categoryQuery := categoryquery.New(categoryRepo)
	favoriteView := favoritequery.New(favRepo, testS3BaseURL)

	// Router (mirrors internal/app/app.go)
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(apimiddleware.Locale)

	router.Get("/health", apihealth.NewHandler(config.Version).ServeHTTP)

	router.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", authhandler.NewHandler(authSvc, config.AuthConfig{RateLimitRPS: 100, RateLimitBurst: 100}, config.AppConfig{}, logger).Routes())
		r.Mount("/user", apiuser.NewHandler(authSvc, userSvc, logger).Routes())
		r.Mount("/upload", uploadapi.NewHandler(mediaSvc, itemSvc, logger).Routes(authSvc))
		r.Mount("/items", itemapi.NewHandler(itemSvc, itemQuery, cityRepo, logger).Routes(authSvc))
		r.Mount("/items/favorites", favoriteapi.NewHandler(favSvc, favoriteView, logger).Routes(authSvc))
		r.Mount("/cities", cityapi.NewHandler(cityRepo, logger).Routes())
		r.Mount("/categories", categoriesapi.NewHandler(categoryQuery, logger).Routes())
		r.Mount("/payments", paymentapi.NewHandler(paymentSvc, "https://example.com/return", cfg.SMTP.FrontendDomain, logger).Routes(authSvc))
		r.Group(func(r chi.Router) {
			r.Use(apimiddleware.RateLimit(cfg.Auth.ContactRateLimitRPS, cfg.Auth.ContactRateLimitBurst))
			r.Mount("/contact-messages", contactapi.NewHandler(contactSvc, logger).Routes())
		})

		// Chat (user <-> user)
		chatRepoInstance := chatrepo.New(tx)
		chatHub := chatapi.NewHub(logger)
		chatSvcInstance := chatservice.New(chatRepoInstance, userRepo, chatHub, testS3BaseURL, logger)
		chatQuerySvc := chatquery.New(chatRepoInstance, testS3BaseURL, userRepo)
		r.Mount("/chat", chatapi.NewHandler(chatSvcInstance, chatQuerySvc, mediaSvc, chatHub, testS3BaseURL, []string{"http://example.com"}, logger).Routes(authSvc))
	})

	return &testApp{
		Handler:     router,
		tx:          tx,
		emailSender: emailSender,
		storage:     storage,
	}
}

// Helper to read verification code from database
func getVerificationCode(t *testing.T, app *testApp, email string) string {
	var code string
	err := app.tx.QueryRow(context.Background(), `SELECT email_verify_code FROM users WHERE email = $1`, email).Scan(&code)
	require.NoError(t, err)
	return code
}

// registerVerifyAndLogin registers a user, verifies email using DB code, and returns access token.
func registerVerifyAndLogin(t *testing.T, app *testApp, email, password string) string {
	return registerVerifyAndLoginTokens(t, app, email, password).AccessToken
}

func registerVerifyAndLoginAsAdmin(t *testing.T, app *testApp, email, password string) string {
	return registerVerifyAndLoginTokensWithRoles(t, app, email, password, []string{"ROLE_ADMIN"}).AccessToken
}

func registerVerifyAndLoginTokens(t *testing.T, app *testApp, email, password string) authhandler.LoginResponse {
	return registerVerifyAndLoginTokensWithRoles(t, app, email, password, nil)
}

// registerVerifyAndLoginTokensWithRoles registers a default ROLE_USER, then (if roles
// is non-empty) overrides the user's roles directly in the DB before login so the
// resulting JWT carries them. Role selection at registration is not supported.
func registerVerifyAndLoginTokensWithRoles(t *testing.T, app *testApp, email, password string, roles []string) authhandler.LoginResponse {
	// Register
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

	if len(roles) > 0 {
		_, err = app.tx.Exec(context.Background(), `UPDATE users SET roles = $1 WHERE email = $2`, roles, email)
		require.NoError(t, err)
	}

	// Verify email using DB code
	code := getVerificationCode(t, app, email)
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

	// Login
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

	var loginResp authhandler.LoginResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&loginResp))
	require.NotEmpty(t, loginResp.AccessToken)
	require.NotEmpty(t, loginResp.RefreshToken)
	return loginResp
}

func extractResetCode(t *testing.T, app *testApp, email string) string {
	var code string
	err := app.tx.QueryRow(context.Background(), `SELECT reset_code FROM users WHERE email = $1`, email).Scan(&code)
	require.NoError(t, err)
	return code
}

// Helper to register user with specific role.
func registerWithRole(t *testing.T, app *testApp, email, password, role string) string {
	if role == "ROLE_ADMIN" {
		return registerVerifyAndLoginAsAdmin(t, app, email, password)
	}
	return registerVerifyAndLogin(t, app, email, password)
}

// uploadItemPhoto uploads a fake JPEG item photo and returns the photo UUID.
func uploadItemPhoto(t *testing.T, app *testApp, token string) uuid.UUID {
	t.Helper()
	body, ct := multipartBody(t, map[string]string{"type": "item"}, "file", "item.jpg", fakeJPEG)
	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	id, err := uuid.Parse(resp["data"]["id"].(string))
	require.NoError(t, err)
	return id
}

// noopPaymentProvider is a stub payment provider for tests; it never calls an external API.
type noopPaymentProvider struct{}

func (n *noopPaymentProvider) CreatePayment(_ context.Context, in provider.CreatePaymentInput) (*provider.CreatedPayment, error) {
	return &provider.CreatedPayment{
		ProviderPaymentID: "test-" + in.LocalPaymentID.String(),
		Status:            "pending",
		ConfirmationURL:   "https://example.com/confirm",
		Metadata:          map[string]any{},
	}, nil
}

func (n *noopPaymentProvider) GetPayment(_ context.Context, providerPaymentID string) (*provider.PaymentInfo, error) {
	return &provider.PaymentInfo{
		ProviderPaymentID: providerPaymentID,
		Status:            model.PaymentStatusPending,
		ProviderStatus:    "pending",
		Metadata:          map[string]any{},
	}, nil
}

func (n *noopPaymentProvider) ParseWebhookEvent(_ []byte) (*provider.WebhookEvent, error) {
	return nil, fmt.Errorf("noop provider does not support webhooks")
}

package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dionisvl/avi/api-go/internal/email"
)

// resendSenderFromEnv creates a ResendSender from env vars.
// Requires: SMTP_PASSWORD (API key), SMTP_FROM, RESEND_TEST_TO
func resendSenderFromEnv(t *testing.T) (email.Sender, string) {
	t.Helper()

	pass := os.Getenv("SMTP_PASSWORD")
	if pass == "" {
		t.Skip("SMTP_PASSWORD not set")
	}

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		t.Skip("SMTP_FROM not set")
	}

	to := os.Getenv("RESEND_TEST_TO")
	if to == "" {
		t.Skip("RESEND_TEST_TO not set")
	}

	frontendDomain := os.Getenv("FRONTEND_DOMAIN")
	if frontendDomain == "" {
		frontendDomain = "localhost:3000"
	}

	// email.New with host="smtp.resend.com" → returns ResendSender (uses REST API)
	sender := email.New("smtp.resend.com", 465, from, "resend", pass, frontendDomain, slog.Default())
	return sender, to
}

// TestResendSender_SendVerificationCode tests sending verification code via Resend REST API.
// Requires: RUN_RESEND_TESTS=true, SMTP_PASSWORD, SMTP_FROM, RESEND_TEST_TO
func TestResendSender_SendVerificationCode(t *testing.T) {
	if os.Getenv("RUN_RESEND_TESTS") != "true" {
		t.Skip("Resend integration tests disabled (set RUN_RESEND_TESTS=true)")
	}

	sender, to := resendSenderFromEnv(t)

	code := "123456"
	t.Logf("Sending verification code to %s via Resend API", to)

	err := sender.SendVerificationCode(context.Background(), email.LocaleRU, to, code)
	require.NoError(t, err)

	t.Logf("OK: verification email sent to %s", to)
	fmt.Printf("Check inbox at %s for subject 'Подтверждение email'\n", to)
}

// TestResendSender_SendPasswordResetCode tests sending password reset code via Resend REST API.
// Requires: RUN_RESEND_TESTS=true, SMTP_PASSWORD, SMTP_FROM, RESEND_TEST_TO
func TestResendSender_SendPasswordResetCode(t *testing.T) {
	if os.Getenv("RUN_RESEND_TESTS") != "true" {
		t.Skip("Resend integration tests disabled (set RUN_RESEND_TESTS=true)")
	}

	sender, to := resendSenderFromEnv(t)

	code := "654321"
	t.Logf("Sending password reset code to %s via Resend API", to)

	err := sender.SendPasswordResetCode(context.Background(), email.LocaleRU, to, code)
	require.NoError(t, err)

	t.Logf("OK: password reset email sent to %s", to)
	fmt.Printf("Check inbox at %s for subject 'Сброс пароля'\n", to)
}

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dionisvl/avi/api-go/internal/email"
)

func checkMailpitConnection(t *testing.T, host string, apiPort int) {
	// Try to connect to Mailpit API on both IPv4 and IPv6
	timeout := 1 * time.Second
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", apiPort))

	// Try TCP dial
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		t.Skipf("Mailpit not reachable at %s - make sure docker-compose is running. Error: %v", addr, err)
		return
	}
	_ = conn.Close()
}

// TestSMTPSender_SendVerificationCode tests sending verification code via SMTP
// Requires: RUN_INTEGRATION_TESTS=true and Mailpit running on SMTP_HOST:SMTP_PORT
func TestSMTPSender_SendVerificationCode(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("integration tests disabled (set RUN_INTEGRATION_TESTS=true)")
	}

	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "localhost"
	}

	smtpPort := 1025
	mailpitAPIPort := 8025

	// Check if Mailpit is reachable
	checkMailpitConnection(t, smtpHost, mailpitAPIPort)

	frontendURL := os.Getenv("FRONTEND_DOMAIN")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	sender := email.NewSMTPSender(smtpHost, smtpPort, "test@avi.app", "", "", frontendURL, slog.Default())

	code := "123456"
	err := sender.SendVerificationCode(context.TODO(), email.LocaleRU, "test-verify@example.com", code)
	require.NoError(t, err)

	// Check Mailpit API for the email
	resp, err := http.Get(fmt.Sprintf("http://%s:%d/api/v1/messages", smtpHost, mailpitAPIPort))
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var messages struct {
		Messages []struct {
			To      []map[string]string `json:"To"`
			Subject string              `json:"Subject"`
			Text    string              `json:"Text"`
			HTML    string              `json:"HTML"`
			Snippet string              `json:"Snippet"`
		} `json:"messages"`
	}

	err = json.Unmarshal(body, &messages)
	require.NoError(t, err)

	// Find our email (last one sent)
	require.Greater(t, len(messages.Messages), 0, "no emails received")

	// Get the most recent email
	email := messages.Messages[len(messages.Messages)-1]
	assert.Equal(t, "test-verify@example.com", email.To[0]["Address"])
	assert.Contains(t, email.Subject, "Подтверждение email")

	// Check body content
	bodyContent := email.Text
	if len(email.Text) == 0 {
		bodyContent = email.HTML
	}
	if len(bodyContent) == 0 {
		bodyContent = email.Snippet
	}

	assert.NotEmpty(t, bodyContent, "email body is empty")
	assert.Contains(t, bodyContent, code)
}

// TestSMTPSender_SendPasswordResetCode tests sending password reset code via SMTP
// Requires: RUN_INTEGRATION_TESTS=true and Mailpit running on SMTP_HOST:SMTP_PORT
func TestSMTPSender_SendPasswordResetCode(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("integration tests disabled (set RUN_INTEGRATION_TESTS=true)")
	}

	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "localhost"
	}

	smtpPort := 1025
	mailpitAPIPort := 8025

	// Check if Mailpit is reachable
	checkMailpitConnection(t, smtpHost, mailpitAPIPort)

	frontendURL := os.Getenv("FRONTEND_DOMAIN")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	sender := email.NewSMTPSender(smtpHost, smtpPort, "test@avi.app", "", "", frontendURL, slog.Default())

	code := "654321"
	err := sender.SendPasswordResetCode(context.TODO(), email.LocaleRU, "test-reset@example.com", code)
	require.NoError(t, err)

	// Check Mailpit API for the email
	resp, err := http.Get(fmt.Sprintf("http://%s:%d/api/v1/messages", smtpHost, mailpitAPIPort))
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var messages struct {
		Messages []struct {
			To      []map[string]string `json:"To"`
			Subject string              `json:"Subject"`
			Text    string              `json:"Text"`
			HTML    string              `json:"HTML"`
			Snippet string              `json:"Snippet"`
		} `json:"messages"`
	}

	err = json.Unmarshal(body, &messages)
	require.NoError(t, err)

	require.Greater(t, len(messages.Messages), 0, "no emails received")

	// Get the most recent email
	email := messages.Messages[len(messages.Messages)-1]
	assert.Equal(t, "test-reset@example.com", email.To[0]["Address"])
	assert.Contains(t, email.Subject, "Сброс пароля")

	// Check body content
	bodyContent := email.Text
	if len(email.Text) == 0 {
		bodyContent = email.HTML
	}
	if len(bodyContent) == 0 {
		bodyContent = email.Snippet
	}

	assert.NotEmpty(t, bodyContent, "email body is empty")
	assert.Contains(t, bodyContent, code)
	assert.Contains(t, bodyContent, "15 минут")
}

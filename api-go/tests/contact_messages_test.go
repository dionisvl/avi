package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactMessages_Create_SendsEmail(t *testing.T) {
	app := newTestApp(t)

	body, err := json.Marshal(map[string]any{
		"name":    "Denis",
		"email":   "denis@example.com",
		"subject": "Partnership",
		"message": "Please contact me about a listing.",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/contact-messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())

	emails := app.emailSender.all()
	require.NotEmpty(t, emails)
	last := emails[len(emails)-1]
	assert.Equal(t, "contact-test@avi.app", last.To)
	assert.Equal(t, "en", last.Locale)
	assert.Contains(t, last.Subject, "Contact form message")
	assert.Contains(t, last.Body, "denis@example.com")
	assert.Contains(t, last.Body, "Please contact me")
}

func TestContactMessages_Create_InvalidEmail(t *testing.T) {
	app := newTestApp(t)

	body, err := json.Marshal(map[string]any{
		"name":    "Denis",
		"email":   "not-email",
		"message": "Please contact me about a listing.",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/contact-messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestContactMessages_Create_RateLimited(t *testing.T) {
	app := newTestApp(t)

	body, err := json.Marshal(map[string]any{
		"name":    "Denis",
		"email":   "denis@example.com",
		"subject": "Partnership",
		"message": "Please contact me about a listing.",
	})
	require.NoError(t, err)

	for range 2 {
		req := httptest.NewRequest("POST", "/api/v1/contact-messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest("POST", "/api/v1/contact-messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

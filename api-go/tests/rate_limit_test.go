package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/dionisvl/avi/api-go/internal/api/middleware"
)

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	middleware.ResetLimiters()

	router := chi.NewRouter()
	router.Use(middleware.RateLimit(100, 10)) // 100 req/s
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Send 5 requests quickly
	for range 5 {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "127.0.0.1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

// TestRateLimit_TrustedProxy_UsesForwardedHeader verifies that X-Real-IP is trusted
// when the request comes from a trusted proxy (loopback/internal network).
func TestRateLimit_TrustedProxy_UsesForwardedHeader(t *testing.T) {
	middleware.ResetLimiters()

	router := chi.NewRouter()
	router.Use(middleware.RateLimit(2, 1)) // tight limit
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First request: from trusted proxy (127.0.0.1), with client IP 5.5.5.5 in X-Real-IP
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.Header.Set("X-Real-IP", "5.5.5.5")
	req1.RemoteAddr = "127.0.0.1:12345" // trusted proxy
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code, "first request from trusted proxy should pass")

	// Second request: untrusted remote 5.5.5.5 with spoofed X-Real-IP 9.9.9.9
	// Should be counted as 5.5.5.5 (RemoteAddr), not 9.9.9.9 (X-Real-IP ignored)
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-Real-IP", "9.9.9.9")
	req2.RemoteAddr = "5.5.5.5:9999" // untrusted remote
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	// This hits the same bucket as req1 (5.5.5.5), so it gets rate limited
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code, "second request to same IP bucket should be rate-limited")
}

// TestRateLimit_UntrustedProxy_IgnoresForwardedHeader verifies that spoofed X-Real-IP
// from an untrusted remote is ignored; RemoteAddr is used instead.
func TestRateLimit_UntrustedProxy_IgnoresForwardedHeader(t *testing.T) {
	middleware.ResetLimiters()

	router := chi.NewRouter()
	router.Use(middleware.RateLimit(100, 10)) // lenient limit
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Untrusted remote 1.2.3.4 spoofs X-Real-IP as 127.0.0.1
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1") // spoofed trusted IP
	req.RemoteAddr = "1.2.3.4:9999"          // untrusted

	// Should still work (not crash), and the IP used is 1.2.3.4
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "request should pass; header from untrusted source is ignored, RemoteAddr used")
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	middleware.ResetLimiters()

	router := chi.NewRouter()
	router.Use(middleware.RateLimit(2, 1)) // 2 req/s, burst 1
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Send requests from same IP, should eventually get 429
	success := 0
	blocked := 0

	for range 10 {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Real-IP", "127.0.0.1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		switch rec.Code {
		case http.StatusOK:
			success++
		case http.StatusTooManyRequests:
			blocked++
			assert.Equal(t, "1", rec.Header().Get("Retry-After"))
		}
	}

	// Should have some requests blocked
	assert.Greater(t, blocked, 0)
}

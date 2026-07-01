package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	apihealth "github.com/dionisvl/avi/api-go/internal/api/health"
	"github.com/dionisvl/avi/api-go/internal/config"
)

func TestHealthEndpoint_Returns200WithCorrectBody(t *testing.T) {
	handler := apihealth.NewHandler("test-v1")

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "test-v1", resp["version"])
}

func TestHealthEndpoint_VersionFromConfig(t *testing.T) {
	// Save original version
	origVersion := config.Version
	defer func() { config.Version = origVersion }()

	config.Version = "1.2.3"
	handler := apihealth.NewHandler(config.Version)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "1.2.3", resp["version"])
}

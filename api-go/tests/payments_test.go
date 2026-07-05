package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPayments_PromoteOwnListing verifies a seller can start a promote_listing
// payment for their own item.
func TestPayments_PromoteOwnListing(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "pay-own-"+uuid.New().String()+"@example.com", "password123")
	itemID := createPublishedItem(t, app, token, "Promote Me")

	body, err := json.Marshal(map[string]any{
		"purpose":    "promote_listing",
		"subject_id": itemID.String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		ID     uuid.UUID `json:"id"`
		Status string    `json:"status"`
		Amount struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		} `json:"amount"`
		ConfirmationURL string `json:"confirmation_url"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Equal(t, "RUB", resp.Amount.Currency)
	assert.NotEmpty(t, resp.ConfirmationURL)
}

// TestPayments_PromoteOthersListingForbidden verifies a user cannot promote a
// listing they do not own.
func TestPayments_PromoteOthersListingForbidden(t *testing.T) {
	app := newTestApp(t)
	ownerToken := registerVerifyAndLogin(t, app, "pay-owner-"+uuid.New().String()+"@example.com", "password123")
	otherToken := registerVerifyAndLogin(t, app, "pay-other-"+uuid.New().String()+"@example.com", "password123")
	itemID := createPublishedItem(t, app, ownerToken, "Not Yours")

	body, err := json.Marshal(map[string]any{
		"purpose":    "promote_listing",
		"subject_id": itemID.String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+otherToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestPayments_DemoCheckoutOthersListingAllowed verifies demo checkout can be
// started for any published listing. It is a payment demo, not seller promotion.
func TestPayments_DemoCheckoutOthersListingAllowed(t *testing.T) {
	app := newTestApp(t)
	ownerToken := registerVerifyAndLogin(t, app, "pay-demo-owner-"+uuid.New().String()+"@example.com", "password123")
	buyerToken := registerVerifyAndLogin(t, app, "pay-demo-buyer-"+uuid.New().String()+"@example.com", "password123")
	itemID := createPublishedItem(t, app, ownerToken, "Demo Checkout")

	body, err := json.Marshal(map[string]any{
		"purpose":    "demo_checkout",
		"subject_id": itemID.String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+buyerToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		ID              uuid.UUID `json:"id"`
		ConfirmationURL string    `json:"confirmation_url"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.NotEmpty(t, resp.ConfirmationURL)
}

// TestPayments_PromoteMissingListingNotFound verifies promoting a non-existent
// item returns 404.
func TestPayments_PromoteMissingListingNotFound(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "pay-missing-"+uuid.New().String()+"@example.com", "password123")

	body, err := json.Marshal(map[string]any{
		"purpose":    "promote_listing",
		"subject_id": uuid.New().String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestPayments_InvalidPurpose verifies an unsupported purpose is rejected.
func TestPayments_InvalidPurpose(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "pay-badpurpose-"+uuid.New().String()+"@example.com", "password123")
	itemID := createPublishedItem(t, app, token, "Bad Purpose")

	body, err := json.Marshal(map[string]any{
		"purpose":    "contact_access",
		"subject_id": itemID.String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestPayments_RequiresAuth verifies the endpoint requires authentication.
func TestPayments_RequiresAuth(t *testing.T) {
	app := newTestApp(t)

	body, err := json.Marshal(map[string]any{
		"purpose":    "promote_listing",
		"subject_id": uuid.New().String(),
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

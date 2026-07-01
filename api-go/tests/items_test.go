package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	itemapi "github.com/dionisvl/avi/api-go/internal/api/item"
)

// Seed reference IDs from internal/migrations/00001_init.sql.
const (
	seedCategoryElectronics = "00000000-0000-0000-0003-000000000001"
	seedCityNewYork         = "00000000-0000-0000-0001-000000000001"
)

// buildCreateItemBody builds a minimal valid POST /items body. The seller is the
// authenticated caller, so no owner is needed.
func buildCreateItemBody(t *testing.T, title string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"title":       title,
		"category_id": seedCategoryElectronics,
		"description": "Test listing for " + title,
		"condition":   "used",
		"city_uuid":   seedCityNewYork,
	})
	require.NoError(t, err)
	return body
}

// buildCreateItemBodyWithPrice is buildCreateItemBody plus a price.
func buildCreateItemBodyWithPrice(t *testing.T, title string, amount int64, currency string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"title":       title,
		"category_id": seedCategoryElectronics,
		"description": "Test listing for " + title,
		"condition":   "used",
		"city_uuid":   seedCityNewYork,
		"price":       map[string]any{"amount": amount, "currency": currency},
	})
	require.NoError(t, err)
	return body
}

// buildCreateItemBodyWithPhotoIDs is buildCreateItemBody plus photo IDs.
func buildCreateItemBodyWithPhotoIDs(t *testing.T, title string, photoIDs []uuid.UUID) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"title":       title,
		"category_id": seedCategoryElectronics,
		"description": "Test listing for " + title,
		"condition":   "used",
		"city_uuid":   seedCityNewYork,
		"photo_ids":   photoIDs,
	})
	require.NoError(t, err)
	return body
}

// createPublishedItem creates an item via the API and returns its ID.
func createPublishedItem(t *testing.T, app *testApp, token, title string) uuid.UUID {
	t.Helper()
	resp := createItem(t, app, token, title)
	return resp.Data.ID
}

// createItem creates an item via the API and returns the decoded response.
func createItem(t *testing.T, app *testApp, token, title string) itemapi.ItemSingleResponse {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewReader(buildCreateItemBody(t, title)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var resp itemapi.ItemSingleResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	return resp
}

func TestItems_List_Public(t *testing.T) {
	app := newTestApp(t)

	// GET /items is public — no auth required.
	req := httptest.NewRequest("GET", "/api/v1/items", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestItems_List_Pagination(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "list-page-"+uuid.New().String()+"@example.com", "password123")

	req := httptest.NewRequest("GET", "/api/v1/items", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp itemapi.ItemListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Pagination.Page)
	assert.Equal(t, 20, resp.Pagination.PerPage)
	assert.GreaterOrEqual(t, resp.Pagination.Total, 0)
}

func TestItems_Create_AnyAuthenticatedUser(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "create-"+uuid.New().String()+"@example.com", "password123")

	resp := createItem(t, app, token, "Comfy Sofa")
	assert.Equal(t, "Comfy Sofa", resp.Data.Title)
	assert.Equal(t, "used", resp.Data.Condition)
	assert.Equal(t, "published", resp.Data.Status)
	assert.NotEqual(t, uuid.Nil, resp.Data.ID)
	assert.NotEmpty(t, resp.Data.Slug)
}

func TestItems_Create_RequiresAuth(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewReader(buildCreateItemBody(t, "No Auth")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestItems_Create_WithPrice(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "create-price-"+uuid.New().String()+"@example.com", "password123")

	req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewReader(buildCreateItemBodyWithPrice(t, "Priced Item", 500000, "RUB")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var resp itemapi.ItemSingleResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.NotNil(t, resp.Data.Price)
	assert.Equal(t, int64(500000), resp.Data.Price.Amount)
	assert.Equal(t, "RUB", resp.Data.Price.Currency)
}

func TestItems_Create_WithPhotoIDs(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "create-photos-"+uuid.New().String()+"@example.com", "password123")

	photoID := uploadItemPhoto(t, app, token)

	body := buildCreateItemBodyWithPhotoIDs(t, "Item With Photo "+uuid.New().String(), []uuid.UUID{photoID})
	req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var resp itemapi.ItemSingleResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Data.Photos, 1)
	assert.Equal(t, photoID, resp.Data.Photos[0].ID)
	assert.NotEmpty(t, resp.Data.Photos[0].URL)
}

func TestItems_GetByID(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "get-"+uuid.New().String()+"@example.com", "password123")
	id := createPublishedItem(t, app, token, "Gettable Item")

	req := httptest.NewRequest("GET", "/api/v1/items/"+id.String(), nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp itemapi.ItemSingleResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, id, resp.Data.ID)
	assert.Equal(t, "Gettable Item", resp.Data.Title)
}

// TestItems_SeedItem_HasExternalPhotos verifies that a demo item seeded with
// absolute placeholder URLs returns those URLs as-is (not joined onto the S3
// base URL). The iPhone demo item has a stable UUID and 3 seeded photos.
func TestItems_SeedItem_HasExternalPhotos(t *testing.T) {
	app := newTestApp(t)

	const iphoneItemID = "00000000-0000-0000-0004-000000000001"
	req := httptest.NewRequest("GET", "/api/v1/items/"+iphoneItemID, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp itemapi.ItemSingleResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	require.Len(t, resp.Data.Photos, 3)
	seen := make(map[string]struct{}, len(resp.Data.Photos))
	for _, p := range resp.Data.Photos {
		assert.True(t, strings.HasPrefix(p.URL, "https://placeholdpicsum.dev/photo/category/"),
			"photo URL must be the external category placeholder as-is, got %q", p.URL)
		assert.Contains(t, p.URL, ".webp", "photo URL must be WebP, got %q", p.URL)
		seen[p.URL] = struct{}{}
	}
	// Each photo in the gallery must have a distinct URL (distinct image).
	assert.Len(t, seen, len(resp.Data.Photos), "gallery photos must be distinct")

	require.NotNil(t, resp.Data.Thumbnail)
	assert.True(t, strings.HasPrefix(resp.Data.Thumbnail.URL, "https://placeholdpicsum.dev/photo/category/"),
		"thumbnail URL must be the external category placeholder as-is, got %q", resp.Data.Thumbnail.URL)
	assert.Contains(t, resp.Data.Thumbnail.URL, ".webp", "thumbnail URL must be WebP")
}

func TestItems_GetByID_NotFound(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/items/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestItems_Update_ByOwner(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "update-"+uuid.New().String()+"@example.com", "password123")
	id := createPublishedItem(t, app, token, "Before Update")

	body, err := json.Marshal(map[string]any{"title": "After Update", "status": "archived"})
	require.NoError(t, err)
	req := httptest.NewRequest("PATCH", "/api/v1/items/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp itemapi.ItemSingleResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "After Update", resp.Data.Title)
	assert.Equal(t, "archived", resp.Data.Status)
}

func TestItems_Update_ByNonOwnerForbidden(t *testing.T) {
	app := newTestApp(t)
	ownerToken := registerVerifyAndLogin(t, app, "owner-"+uuid.New().String()+"@example.com", "password123")
	otherToken := registerVerifyAndLogin(t, app, "other-"+uuid.New().String()+"@example.com", "password123")
	id := createPublishedItem(t, app, ownerToken, "Someone Elses Item")

	body, err := json.Marshal(map[string]any{"title": "Hijacked"})
	require.NoError(t, err)
	req := httptest.NewRequest("PATCH", "/api/v1/items/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+otherToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestItems_Delete_ByOwner(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "delete-"+uuid.New().String()+"@example.com", "password123")
	id := createPublishedItem(t, app, token, "To Delete")

	req := httptest.NewRequest("DELETE", "/api/v1/items/"+id.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Confirm gone.
	getReq := httptest.NewRequest("GET", "/api/v1/items/"+id.String(), nil)
	getRec := httptest.NewRecorder()
	app.ServeHTTP(getRec, getReq)
	assert.Equal(t, http.StatusNotFound, getRec.Code)
}

func TestItems_Filter_Search(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "search-"+uuid.New().String()+"@example.com", "password123")
	unique := "Zxqw" + uuid.New().String()[:8]
	createPublishedItem(t, app, token, unique+" Gadget")

	req := httptest.NewRequest("GET", "/api/v1/items?search="+unique, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp itemapi.ItemListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Pagination.Total)
}

func TestItems_Filter_Condition(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/items?condition=used", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

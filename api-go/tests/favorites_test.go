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

	favoriteapi "github.com/dionisvl/avi/api-go/internal/api/favorite"
)

func TestFavorites_List_Empty(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "fav-list-empty@example.com", "password123")

	req := httptest.NewRequest("GET", "/api/v1/items/favorites", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp favoriteapi.FavoritesListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Empty(t, resp.Data)
	assert.Equal(t, 0, resp.Pagination.Total)
}

func TestFavorites_List_RequiresAuth(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/items/favorites", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFavorites_Add_And_List(t *testing.T) {
	app := newTestApp(t)
	ownerToken := registerVerifyAndLogin(t, app, "fav-owner-"+uuid.New().String()+"@example.com", "password123")
	favoriterToken := registerVerifyAndLogin(t, app, "fav-user-"+uuid.New().String()+"@example.com", "password123")

	// Create an item first
	itemID := createPublishedItem(t, app, ownerToken, "FavItem")

	// Add to favorites
	addBody, err := json.Marshal(map[string]string{"item_id": itemID.String()})
	require.NoError(t, err)
	addReq := httptest.NewRequest("POST", "/api/v1/items/favorites", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	addRec := httptest.NewRecorder()
	app.ServeHTTP(addRec, addReq)

	assert.Equal(t, http.StatusCreated, addRec.Code)

	// List favorites — should contain the item
	listReq := httptest.NewRequest("GET", "/api/v1/items/favorites", nil)
	listReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	listRec := httptest.NewRecorder()
	app.ServeHTTP(listRec, listReq)

	assert.Equal(t, http.StatusOK, listRec.Code)

	var resp favoriteapi.FavoritesListResponse
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Pagination.Total)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, itemID, resp.Data[0].ItemID)
	assert.Equal(t, "FavItem", resp.Data[0].Item.Title)
}

func TestFavorites_Add_Duplicate_409(t *testing.T) {
	app := newTestApp(t)
	ownerToken := registerVerifyAndLogin(t, app, "fav-dup-owner-"+uuid.New().String()+"@example.com", "password123")
	favoriterToken := registerVerifyAndLogin(t, app, "fav-dup-"+uuid.New().String()+"@example.com", "password123")

	itemID := createPublishedItem(t, app, ownerToken, "DupItem")

	addBody, err := json.Marshal(map[string]string{"item_id": itemID.String()})
	require.NoError(t, err)

	// First add — success
	addReq := httptest.NewRequest("POST", "/api/v1/items/favorites", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	addRec := httptest.NewRecorder()
	app.ServeHTTP(addRec, addReq)
	assert.Equal(t, http.StatusCreated, addRec.Code)

	// Second add — conflict
	addBody2, err := json.Marshal(map[string]string{"item_id": itemID.String()})
	require.NoError(t, err)
	addReq2 := httptest.NewRequest("POST", "/api/v1/items/favorites", bytes.NewReader(addBody2))
	addReq2.Header.Set("Content-Type", "application/json")
	addReq2.Header.Set("Authorization", "Bearer "+favoriterToken)
	addRec2 := httptest.NewRecorder()
	app.ServeHTTP(addRec2, addReq2)

	assert.Equal(t, http.StatusConflict, addRec2.Code)
}

func TestFavorites_Delete_Success(t *testing.T) {
	app := newTestApp(t)
	ownerToken := registerVerifyAndLogin(t, app, "fav-del-owner-"+uuid.New().String()+"@example.com", "password123")
	favoriterToken := registerVerifyAndLogin(t, app, "fav-del-"+uuid.New().String()+"@example.com", "password123")

	itemID := createPublishedItem(t, app, ownerToken, "DelFavItem")

	// Add
	addBody, err := json.Marshal(map[string]string{"item_id": itemID.String()})
	require.NoError(t, err)
	addReq := httptest.NewRequest("POST", "/api/v1/items/favorites", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	app.ServeHTTP(httptest.NewRecorder(), addReq)

	// Delete
	delReq := httptest.NewRequest("DELETE", "/api/v1/items/favorites/"+itemID.String(), nil)
	delReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	delRec := httptest.NewRecorder()
	app.ServeHTTP(delRec, delReq)

	assert.Equal(t, http.StatusNoContent, delRec.Code)

	// List should be empty
	listReq := httptest.NewRequest("GET", "/api/v1/items/favorites", nil)
	listReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	listRec := httptest.NewRecorder()
	app.ServeHTTP(listRec, listReq)

	var resp favoriteapi.FavoritesListResponse
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Pagination.Total)
}

func TestFavorites_Delete_NotFound_404(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "fav-del-notfound@example.com", "password123")

	delReq := httptest.NewRequest("DELETE", "/api/v1/items/favorites/"+uuid.New().String(), nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delRec := httptest.NewRecorder()
	app.ServeHTTP(delRec, delReq)

	assert.Equal(t, http.StatusNotFound, delRec.Code)
}

func TestFavorites_Add_InvalidBody(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "fav-invalid@example.com", "password123")

	addReq := httptest.NewRequest("POST", "/api/v1/items/favorites", bytes.NewReader([]byte(`{}`)))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+token)
	addRec := httptest.NewRecorder()
	app.ServeHTTP(addRec, addReq)

	assert.Equal(t, http.StatusBadRequest, addRec.Code)
}

func TestFavorites_List_ContainsSlug(t *testing.T) {
	app := newTestApp(t)
	ownerToken := registerVerifyAndLogin(t, app, "fav-slug-owner-"+uuid.New().String()+"@example.com", "password123")
	favoriterToken := registerVerifyAndLogin(t, app, "fav-slug-"+uuid.New().String()+"@example.com", "password123")

	created := createItem(t, app, ownerToken, "SlugItem")
	itemID := created.Data.ID
	itemSlug := created.Data.Slug
	require.NotEmpty(t, itemSlug)

	addBody, err := json.Marshal(map[string]string{"item_id": itemID.String()})
	require.NoError(t, err)
	addReq := httptest.NewRequest("POST", "/api/v1/items/favorites", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	app.ServeHTTP(httptest.NewRecorder(), addReq)

	listReq := httptest.NewRequest("GET", "/api/v1/items/favorites", nil)
	listReq.Header.Set("Authorization", "Bearer "+favoriterToken)
	listRec := httptest.NewRecorder()
	app.ServeHTTP(listRec, listReq)

	assert.Equal(t, http.StatusOK, listRec.Code)

	var resp favoriteapi.FavoritesListResponse
	require.NoError(t, json.NewDecoder(listRec.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.NotEmpty(t, resp.Data[0].Item.Slug)
	assert.Equal(t, itemSlug, resp.Data[0].Item.Slug)
}

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeJPEG is a minimal valid JPEG header that passes http.DetectContentType.
var fakeJPEG = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 508)...)

func multipartBody(t *testing.T, fields map[string]string, fileField, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	fw, err := w.CreateFormFile(fileField, filename)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func TestUpload_Avatar_Success(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "uploader-"+uuid.New().String()+"@example.com", "password123")

	body, ct := multipartBody(t, map[string]string{"type": "avatar"}, "file", "avatar.jpg", fakeJPEG)

	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response must have 'data' object")
	assert.NotEmpty(t, data["id"])
	assert.NotEmpty(t, data["url"])
	assert.NotEmpty(t, data["thumbnail_url"])
}

func TestUpload_ItemPhoto_Success(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "item-upload-"+uuid.New().String()+"@example.com", "password123")

	// Create an item first to get a real item_id
	itemID := createPublishedItem(t, app, token, "UploadItem").String()

	body, ct := multipartBody(t, map[string]string{"type": "item", "item_id": itemID}, "file", "item.jpg", fakeJPEG)

	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.NotEmpty(t, data["url"])
}

func TestUpload_ItemPhoto_WithoutItemID(t *testing.T) {
	// item_id is now optional: photos can be uploaded before the item is created
	// and linked later via photo_ids in POST /items.
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "item-upload-noid@example.com", "password123")

	body, ct := multipartBody(t, map[string]string{"type": "item"}, "file", "item.jpg", fakeJPEG)

	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestUpload_NoAuth(t *testing.T) {
	app := newTestApp(t)
	body, ct := multipartBody(t, map[string]string{"type": "avatar"}, "file", "avatar.jpg", []byte("data"))

	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpload_InvalidType(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "type-test@example.com", "password123")

	body, ct := multipartBody(t, map[string]string{"type": "document"}, "file", "file.pdf", []byte("data"))

	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpload_FileTooLarge(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "size-test@example.com", "password123")

	// 6 MB > 5 MB limit
	largeContent := make([]byte, 6<<20)
	body, ct := multipartBody(t, map[string]string{"type": "avatar"}, "file", "big.jpg", largeContent)

	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpload_MissingFile(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "nofile@example.com", "password123")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("type", "avatar"))
	require.NoError(t, w.Close())

	req := httptest.NewRequest("POST", "/api/v1/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMe_AvatarURL_AfterUpload(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "me-avatar@example.com", "password123")

	// Upload avatar first
	body, ct := multipartBody(t, map[string]string{"type": "avatar"}, "file", "avatar.jpg", fakeJPEG)
	req := httptest.NewRequest("POST", "/api/v1/upload", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var uploadResp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&uploadResp))
	uploadedURL := uploadResp["data"].(map[string]any)["url"].(string)

	// /user/me should return avatar_url
	req = httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var meResp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&meResp))
	assert.Equal(t, uploadedURL, meResp["avatar_url"],
		fmt.Sprintf("avatar_url in /user/me should match uploaded URL, got: %v", meResp["avatar_url"]))
}

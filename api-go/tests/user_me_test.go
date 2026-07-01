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

	apiuser "github.com/dionisvl/avi/api-go/internal/api/user"
)

func TestUserMe_GetMe_AllFields(t *testing.T) {
	app := newTestApp(t)
	email := "getme-fields@example.com"
	token := registerVerifyAndLogin(t, app, email, "password123")

	req := httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp apiuser.MeResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, email, resp.Email)
	assert.NotEmpty(t, resp.ID)
	assert.NotEmpty(t, resp.Roles)
	assert.False(t, resp.HasProfile)   // name is empty → no profile yet
	assert.True(t, resp.EmailVerified) // email verified in registerVerifyAndLogin
	assert.False(t, resp.CreatedAt.IsZero())
}

func TestUserMe_UpdateMe_NameOnly(t *testing.T) {
	app := newTestApp(t)
	email := "patch-me-" + uuid.New().String() + "@example.com"
	token := registerVerifyAndLogin(t, app, email, "password123")

	patchBody, err := json.Marshal(map[string]string{
		"name": "John Doe",
	})
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", "/api/v1/user/me", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp apiuser.MeResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "John Doe", resp.Name)
	assert.True(t, resp.HasProfile)
}

func TestUserMe_UpdateMe_OnlyName(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "patch-name-only-"+uuid.New().String()+"@example.com", "password123")

	patchBody, err := json.Marshal(map[string]string{"name": "Jane"})
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", "/api/v1/user/me", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp apiuser.MeResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Jane", resp.Name)
	assert.True(t, resp.HasProfile)
}

func TestUserMe_UpdateMe_EmptyPatch_400(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "patch-noname-"+uuid.New().String()+"@example.com", "password123")

	patchBody, err := json.Marshal(map[string]string{})
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", "/api/v1/user/me", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserMe_UpdateMe_NameNull_ClearsProfileName(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "patch-clear-name-"+uuid.New().String()+"@example.com", "password123")

	patchBody, err := json.Marshal(map[string]string{"name": "Jane"})
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", "/api/v1/user/me", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	patchBody, err = json.Marshal(map[string]any{"name": nil})
	require.NoError(t, err)

	req = httptest.NewRequest("PATCH", "/api/v1/user/me", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp apiuser.MeResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "", resp.Name)
	assert.False(t, resp.HasProfile)
}

func TestUserMe_UpdateMe_KeepsUploadedAvatar(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "patch-keep-avatar-"+uuid.New().String()+"@example.com", "password123")

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

	patchBody, err := json.Marshal(map[string]string{"name": "Avatar Owner"})
	require.NoError(t, err)

	req = httptest.NewRequest("PATCH", "/api/v1/user/me", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp apiuser.MeResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Avatar Owner", resp.Name)
	assert.Equal(t, uploadedURL, resp.AvatarURL)
}

func TestUserMe_DeleteMe_WrongPassword_401(t *testing.T) {
	app := newTestApp(t)
	token := registerVerifyAndLogin(t, app, "del-wrongpwd@example.com", "password123")

	delBody, err := json.Marshal(map[string]string{"password": "wrongpassword"})
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/v1/user/me", bytes.NewReader(delBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// ErrInvalidCredentials → 401
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserMe_DeleteMe_CorrectPassword_204(t *testing.T) {
	app := newTestApp(t)
	password := "password123"
	tokens := registerVerifyAndLoginTokens(t, app, "del-correct@example.com", password)

	uploadBody, ct := multipartBody(t, map[string]string{"type": "avatar"}, "file", "avatar.jpg", fakeJPEG)
	uploadReq := httptest.NewRequest("POST", "/api/v1/upload", uploadBody)
	uploadReq.Header.Set("Content-Type", ct)
	uploadReq.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	uploadRec := httptest.NewRecorder()
	app.ServeHTTP(uploadRec, uploadReq)
	require.Equal(t, http.StatusCreated, uploadRec.Code, "response: %s", uploadRec.Body.String())

	var uploadResp map[string]any
	require.NoError(t, json.NewDecoder(uploadRec.Body).Decode(&uploadResp))
	uploadedURL := uploadResp["data"].(map[string]any)["url"].(string)
	uploadedObjectKey := strings.TrimPrefix(uploadedURL, testS3BaseURL+"/")

	delBody, err := json.Marshal(map[string]string{"password": password})
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/v1/user/me", bytes.NewReader(delBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Contains(t, app.storage.deletedObjectKeys(), uploadedObjectKey)
}

func TestUserMe_DeleteMe_InvalidatesOldAccessAndRefreshTokens(t *testing.T) {
	app := newTestApp(t)
	password := "password123"
	tokens := registerVerifyAndLoginTokens(t, app, "del-revoke@example.com", password)

	delBody, err := json.Marshal(map[string]string{"password": password})
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/v1/user/me", bytes.NewReader(delBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)

	req = httptest.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	refreshBody, err := json.Marshal(map[string]string{"refresh_token": tokens.RefreshToken})
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserMe_DeleteMe_RequiresAuth(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("DELETE", "/api/v1/user/me", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

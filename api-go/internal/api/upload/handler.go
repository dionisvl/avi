package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	mediaservice "github.com/dionisvl/avi/api-go/internal/service/media"
)

const (
	maxUploadSize  = 5 << 20                 // 5 MB parse buffer
	maxFileSize    = 5 << 20                 // 5 MB validated limit
	maxRequestSize = maxFileSize + (1 << 20) // file limit + 1 MB slack for multipart headers and other form fields
)

type UploadType string

const (
	UploadTypeItem   UploadType = "item"
	UploadTypeAvatar UploadType = "avatar"
)

func (t UploadType) valid() bool {
	return t == UploadTypeItem || t == UploadTypeAvatar
}

// ItemOwnerChecker is used to verify that a user has management rights over an item.
type ItemOwnerChecker interface {
	CanManage(ctx context.Context, itemID uuid.UUID, userID uuid.UUID, isAdmin bool) (bool, error)
}

type Handler struct {
	svc       mediaservice.Service
	itemCheck ItemOwnerChecker
	logger    *slog.Logger
}

func NewHandler(svc mediaservice.Service, itemCheck ItemOwnerChecker, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, itemCheck: itemCheck, logger: logger}
}

func (h *Handler) Routes(authSvc apimiddleware.TokenValidator) chi.Router {
	r := chi.NewRouter()
	r.Use(apimiddleware.AuthRequired(authSvc))
	r.Post("/", h.upload)
	return r
}

// @Summary Upload a file
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param type formData string true "Upload target" Enums(item, avatar)
// @Param item_id formData string false "Optional item UUID when type=item; if omitted, the upload is created without an associated item"
// @Param file formData file true "Image file"
// @Success 201 {object} UploadResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/upload [post]
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		api.ValidationError(w, h.logger, "file too large or invalid multipart form")
		return
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}

	uploadType := UploadType(r.FormValue("type"))
	if !uploadType.valid() {
		api.ValidationError(w, h.logger, "type must be 'item' or 'avatar'")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		api.ValidationError(w, h.logger, "missing 'file' field in form")
		return
	}
	defer file.Close() //nolint:errcheck

	if header.Size > maxFileSize {
		api.ValidationError(w, h.logger, "file size exceeds 5 MB limit")
		return
	}

	detectedMime, body, err := sniffImageMime(file)
	if err != nil {
		api.ValidationError(w, h.logger, "unsupported image type: only jpeg, png, webp, gif are allowed")
		return
	}

	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	var result *mediaservice.UploadResult

	switch uploadType {
	case UploadTypeItem:
		var itemID *uuid.UUID
		if rawID := r.FormValue("item_id"); rawID != "" {
			parsed, err := uuid.Parse(rawID)
			if err != nil {
				api.ValidationError(w, h.logger, "invalid item_id")
				return
			}
			roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)
			isAdmin := slices.Contains(roles, string(model.RoleAdmin))
			ok, err := h.itemCheck.CanManage(r.Context(), parsed, userID, isAdmin)
			if err != nil {
				api.Error(w, h.logger, apierr.New(apierr.ErrInternal, "failed to verify item ownership"))
				return
			}
			if !ok {
				api.Error(w, h.logger, apierr.New(apierr.ErrForbidden, "you do not have permission to upload photos for this item"))
				return
			}
			itemID = &parsed
		}

		result, err = h.svc.UploadItemPhoto(r.Context(), mediaservice.UploadItemPhotoInput{
			UploaderID:       userID,
			ItemID:           itemID,
			MimeType:         detectedMime,
			SizeBytes:        header.Size,
			OriginalFilename: header.Filename,
			Body:             body,
		})

	case UploadTypeAvatar:
		result, err = h.svc.UploadUserAvatar(r.Context(), mediaservice.UploadUserAvatarInput{
			UserID:           userID,
			MimeType:         detectedMime,
			SizeBytes:        header.Size,
			OriginalFilename: header.Filename,
			Body:             body,
		})
	}

	if err != nil {
		h.logger.Error("upload failed", slog.String("type", string(uploadType)), slog.String("error", err.Error()))
		api.Error(w, h.logger, apierr.New(apierr.ErrInternal, "Upload failed"))
		return
	}

	api.JSON(w, http.StatusCreated, map[string]any{
		"data": UploadData{
			ID:           result.ID,
			URL:          result.URL,
			ThumbnailURL: result.URL, // TODO: generate real thumbnail
		},
	})
}

type UploadResponse struct {
	Data UploadData `json:"data"`
}

type UploadData struct {
	ID           uuid.UUID `json:"id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
}

// allowedImageMimeTypes is the whitelist of image formats we accept for uploads.
// Keep in sync with extFromMime in internal/service/media.
var allowedImageMimeTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

// sniffImageMime buffers r fully, detects the content type from the first
// 512 bytes, validates it against the image whitelist, and returns the
// detected type along with a seekable reader over the full content. A
// seekable body is required so the S3 SDK can set Content-Length and retry.
func sniffImageMime(r io.Reader) (string, *bytes.Reader, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return "", nil, err
	}

	head := buf
	if len(head) > 512 {
		head = head[:512]
	}

	ct := http.DetectContentType(head)
	if _, ok := allowedImageMimeTypes[ct]; !ok {
		return "", nil, errUnsupportedImageType
	}

	return ct, bytes.NewReader(buf), nil
}

var errUnsupportedImageType = errors.New("unsupported image type")

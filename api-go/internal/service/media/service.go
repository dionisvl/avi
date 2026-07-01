package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dionisvl/avi/api-go/internal/imageproc"
	"github.com/dionisvl/avi/api-go/internal/model"
	mediarepo "github.com/dionisvl/avi/api-go/internal/repository/media"
)

func (s *service) GetUserAvatarURL(ctx context.Context, userID uuid.UUID) string {
	_, objectKey, err := s.repo.GetUserAvatarObjectKey(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ""
	}
	if err != nil {
		s.logger.Error("failed to get user avatar", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
		return ""
	}
	return fmt.Sprintf("%s/%s", s.endpoint, objectKey)
}

func extFromMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "bin"
	}
}

type UploadItemPhotoInput struct {
	UploaderID       uuid.UUID
	ItemID           *uuid.UUID // nil when uploading before item is created
	MimeType         string
	SizeBytes        int64
	OriginalFilename string
	Body             io.Reader
}

type UploadUserAvatarInput struct {
	UserID           uuid.UUID
	MimeType         string
	SizeBytes        int64
	OriginalFilename string
	Body             io.Reader
}

type UploadChatAttachmentInput struct {
	ConversationID uuid.UUID
	Body           io.Reader
}

type UploadResult struct {
	ID        uuid.UUID
	ObjectKey string
	URL       string
	Size      int64
}

type Service interface {
	UploadItemPhoto(ctx context.Context, in UploadItemPhotoInput) (*UploadResult, error)
	UploadUserAvatar(ctx context.Context, in UploadUserAvatarInput) (*UploadResult, error)
	UploadChatAttachment(ctx context.Context, in UploadChatAttachmentInput) (*UploadResult, error)
	GetUserAvatarURL(ctx context.Context, userID uuid.UUID) string
	GetUserAvatarObjectKey(ctx context.Context, userID uuid.UUID) (string, error)
	DeleteObject(ctx context.Context, objectKey string) error
}

type service struct {
	repo      mediarepo.Repository
	s3        objectStorage
	bucket    string
	endpoint  string
	logger    *slog.Logger
	converter *imageproc.Converter
}

type objectStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) (string, error)
	Delete(ctx context.Context, key string) error
}

func New(repo mediarepo.Repository, s3 objectStorage, bucket, endpoint string, logger *slog.Logger) Service {
	return &service{
		repo:      repo,
		s3:        s3,
		bucket:    bucket,
		endpoint:  endpoint,
		logger:    logger,
		converter: imageproc.New(imageproc.Options{}),
	}
}

func (s *service) UploadItemPhoto(ctx context.Context, in UploadItemPhotoInput) (*UploadResult, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	now := time.Now()
	objectKey := fmt.Sprintf("items/pictures/%d/%02d/%02d/%s/orig.%s",
		now.Year(), now.Month(), now.Day(), id.String(), extFromMime(in.MimeType))

	if _, err := s.s3.Upload(ctx, objectKey, in.MimeType, in.Body, in.SizeBytes); err != nil {
		return nil, fmt.Errorf("s3 upload: %w", err)
	}

	uploaderID := in.UploaderID
	p := &model.ItemPhoto{
		ID:               id,
		ItemID:           in.ItemID, // *uuid.UUID, nil = uploaded before item creation
		UploaderID:       &uploaderID,
		Bucket:           s.bucket,
		ObjectKey:        objectKey,
		MimeType:         in.MimeType,
		SizeBytes:        in.SizeBytes,
		OriginalFilename: in.OriginalFilename,
		SortOrder:        0,
		CreatedAt:        now,
	}
	if err := s.repo.CreateItemPhoto(ctx, p); err != nil {
		return nil, fmt.Errorf("save item photo: %w", err)
	}

	return &UploadResult{
		ID:        id,
		ObjectKey: objectKey,
		URL:       fmt.Sprintf("%s/%s", s.endpoint, objectKey),
	}, nil
}

func (s *service) UploadUserAvatar(ctx context.Context, in UploadUserAvatarInput) (*UploadResult, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	now := time.Now()
	objectKey := fmt.Sprintf("users/pictures/%d/%02d/%02d/%s/orig.%s",
		now.Year(), now.Month(), now.Day(), id.String(), extFromMime(in.MimeType))

	if _, err := s.s3.Upload(ctx, objectKey, in.MimeType, in.Body, in.SizeBytes); err != nil {
		return nil, fmt.Errorf("s3 upload: %w", err)
	}

	a := &model.UserAvatar{
		ID:               id,
		UserID:           in.UserID,
		Bucket:           s.bucket,
		ObjectKey:        objectKey,
		MimeType:         in.MimeType,
		SizeBytes:        in.SizeBytes,
		OriginalFilename: in.OriginalFilename,
		CreatedAt:        time.Now(),
	}
	if err := s.repo.UpsertUserAvatar(ctx, a); err != nil {
		return nil, fmt.Errorf("save user avatar: %w", err)
	}

	return &UploadResult{
		ID:        id,
		ObjectKey: objectKey,
		URL:       fmt.Sprintf("%s/%s", s.endpoint, objectKey),
	}, nil
}

func (s *service) GetUserAvatarObjectKey(ctx context.Context, userID uuid.UUID) (string, error) {
	_, objectKey, err := s.repo.GetUserAvatarObjectKey(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get user avatar object: %w", err)
	}
	return objectKey, nil
}

func (s *service) UploadChatAttachment(ctx context.Context, in UploadChatAttachmentInput) (*UploadResult, error) {
	result, err := s.converter.ConvertReader(in.Body)
	if err != nil {
		return nil, fmt.Errorf("convert image: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	objectKey := fmt.Sprintf("chat/%s/%s/conv.webp", in.ConversationID.String(), id.String())

	size := int64(len(result.Data))
	if _, err := s.s3.Upload(ctx, objectKey, "image/webp", bytes.NewReader(result.Data), size); err != nil {
		return nil, fmt.Errorf("s3 upload: %w", err)
	}

	return &UploadResult{
		ID:        id,
		ObjectKey: objectKey,
		URL:       fmt.Sprintf("%s/%s", s.endpoint, objectKey),
		Size:      size,
	}, nil
}

func (s *service) DeleteObject(ctx context.Context, objectKey string) error {
	if err := s.s3.Delete(ctx, objectKey); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

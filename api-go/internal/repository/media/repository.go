package media

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
)

type Repository interface {
	CreateItemPhoto(ctx context.Context, p *model.ItemPhoto) error
	UpsertUserAvatar(ctx context.Context, a *model.UserAvatar) error
	GetUserAvatarObjectKey(ctx context.Context, userID uuid.UUID) (bucket, objectKey string, err error)
	SetItemPhotoItemID(ctx context.Context, photoID uuid.UUID, itemID uuid.UUID) error
	ReplaceItemPhotos(ctx context.Context, itemID uuid.UUID, photoIDs []uuid.UUID) error
	GetItemPhotoUploaderIDs(ctx context.Context, photoIDs []uuid.UUID) (map[uuid.UUID]*uuid.UUID, error)
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateItemPhoto(ctx context.Context, p *model.ItemPhoto) error {
	query := `
		INSERT INTO item_photos (id, item_id, uploader_id, bucket, object_key, mime_type, size_bytes, original_filename, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		p.ID, p.ItemID, p.UploaderID, p.Bucket, p.ObjectKey,
		p.MimeType, p.SizeBytes, p.OriginalFilename, p.SortOrder, p.CreatedAt,
	)
	return err
}

func (r *repository) GetItemPhotoUploaderIDs(ctx context.Context, photoIDs []uuid.UUID) (map[uuid.UUID]*uuid.UUID, error) {
	if len(photoIDs) == 0 {
		return map[uuid.UUID]*uuid.UUID{}, nil
	}
	query := `SELECT id, uploader_id FROM item_photos WHERE id = ANY($1)`
	rows, err := r.db.Query(ctx, query, photoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]*uuid.UUID, len(photoIDs))
	for rows.Next() {
		var photoID uuid.UUID
		var uploaderID *uuid.UUID
		if err := rows.Scan(&photoID, &uploaderID); err != nil {
			return nil, err
		}
		result[photoID] = uploaderID
	}
	return result, rows.Err()
}

func (r *repository) GetUserAvatarObjectKey(ctx context.Context, userID uuid.UUID) (bucket, objectKey string, err error) {
	query := `SELECT bucket, object_key FROM user_avatars WHERE user_id = $1`
	err = r.db.QueryRow(ctx, query, userID).Scan(&bucket, &objectKey)
	return bucket, objectKey, err
}

func (r *repository) SetItemPhotoItemID(ctx context.Context, photoID uuid.UUID, itemID uuid.UUID) error {
	query := `UPDATE item_photos SET item_id = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, itemID, photoID)
	return err
}

func (r *repository) ReplaceItemPhotos(ctx context.Context, itemID uuid.UUID, photoIDs []uuid.UUID) error {
	detachQuery := `
		UPDATE item_photos
		SET item_id = NULL
		WHERE item_id = $1
		  AND NOT (id = ANY($2::uuid[]))
	`
	if _, err := r.db.Exec(ctx, detachQuery, itemID, photoIDs); err != nil {
		return err
	}
	if len(photoIDs) == 0 {
		return nil
	}

	attachQuery := `
		UPDATE item_photos AS ap
		SET item_id = $1,
		    sort_order = ordered.sort_order
		FROM (
			SELECT photo_id, (position - 1)::smallint AS sort_order
			FROM unnest($2::uuid[]) WITH ORDINALITY AS input(photo_id, position)
		) AS ordered
		WHERE ap.id = ordered.photo_id
	`
	tag, err := r.db.Exec(ctx, attachQuery, itemID, photoIDs)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != int64(len(photoIDs)) {
		return fmt.Errorf("expected to attach %d photos, affected %d", len(photoIDs), tag.RowsAffected())
	}
	return nil
}

func (r *repository) UpsertUserAvatar(ctx context.Context, a *model.UserAvatar) error {
	query := `
		INSERT INTO user_avatars (id, user_id, bucket, object_key, mime_type, size_bytes, original_filename, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			id                = EXCLUDED.id,
			bucket            = EXCLUDED.bucket,
			object_key        = EXCLUDED.object_key,
			mime_type         = EXCLUDED.mime_type,
			size_bytes        = EXCLUDED.size_bytes,
			original_filename = EXCLUDED.original_filename,
			created_at        = $8
	`
	_, err := r.db.Exec(ctx, query,
		a.ID, a.UserID, a.Bucket, a.ObjectKey,
		a.MimeType, a.SizeBytes, a.OriginalFilename, time.Now(),
	)
	return err
}

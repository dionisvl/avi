package favorite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
	"github.com/dionisvl/avi/api-go/internal/platform/locale"
	"github.com/dionisvl/avi/api-go/internal/platform/pagination"
)

type ListResult struct {
	Items      []model.FavoriteWithItem
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

type Repository interface {
	List(ctx context.Context, userID uuid.UUID, page, perPage int) (*ListResult, error)
	Add(ctx context.Context, userID, itemID uuid.UUID) error
	Delete(ctx context.Context, userID, itemID uuid.UUID) error
	Exists(ctx context.Context, userID, itemID uuid.UUID) (bool, error)
	// ExistsSet returns the subset of itemIDs that are favorited by userID.
	ExistsSet(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type repository struct {
	db dbtx.DB
}

const (
	pgErrCodeForeignKeyViolation = "23503"
	pgErrCodeUniqueViolation     = "23505"
)

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) List(ctx context.Context, userID uuid.UUID, page, perPage int) (*ListResult, error) {
	pp := pagination.NewParams(page, perPage)
	loc := locale.FromCtx(ctx)

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_favorites WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count favorites: %w", err)
	}

	query := `
		SELECT
			f.id, f.user_id, f.item_id, f.created_at,
			i.id, i.seller_id, i.created_by, i.slug, i.title, i.category_id, i.description, i.tags, i.city_id, i.condition, i.status, i.price_amount, i.price_currency, i.thumbnail_id, i.created_at, i.updated_at,
			u.id, u.name,
			ic.id, ic.slug, ic.parent_id, ic.names, ic.sort_order, ic.is_active,
			icc.slug, icc.geoname_id, icc.names, icc.is_active, icc.population
		FROM user_favorites f
		JOIN items i ON i.id = f.item_id
		JOIN users u ON u.id = i.seller_id
		JOIN categories ic ON ic.id = i.category_id
		JOIN cities icc ON icc.id = i.city_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, pp.Limit(), pp.Offset())
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()

	var items []model.FavoriteWithItem
	var itemIDs []uuid.UUID
	for rows.Next() {
		var fav model.FavoriteWithItem
		it := &fav.Item
		it.Category = &model.Category{} // Item.Category is a pointer; scanned into below
		var itemDesc *string
		var tags []string
		var categoryNamesJSON []byte
		var categoryParentID *uuid.UUID
		var cityNamesJSON []byte
		var sellerName *string
		var priceAmount *int64
		var priceCurrency *string

		err := rows.Scan(
			&fav.ID, &fav.UserID, &fav.ItemID, &fav.CreatedAt,
			&it.ID, &it.SellerID, &it.CreatedBy, &it.Slug, &it.Title, &it.CategoryID, &itemDesc, &tags, &it.City.ID, &it.Condition, &it.Status, &priceAmount, &priceCurrency, &it.ThumbnailID, &it.CreatedAt, &it.UpdatedAt,
			&it.Seller.ID, &sellerName,
			&it.Category.ID, &it.Category.Slug, &categoryParentID, &categoryNamesJSON, &it.Category.SortOrder, &it.Category.IsActive,
			&it.City.Slug, &it.City.GeonameID, &cityNamesJSON, &it.City.IsActive, &it.City.Population,
		)
		if err != nil {
			return nil, fmt.Errorf("scan favorite row: %w", err)
		}

		if itemDesc != nil {
			it.Description = *itemDesc
		}
		if len(tags) > 0 {
			it.Tags = &tags
		}

		// Unmarshal category names and resolve localized name
		if err := json.Unmarshal(categoryNamesJSON, &it.Category.Names); err != nil {
			return nil, fmt.Errorf("unmarshal favorite category names: %w", err)
		}
		it.Category.ParentID = categoryParentID
		it.Category.Name = it.Category.LocalizedName(loc)

		// Unmarshal city names
		if err := json.Unmarshal(cityNamesJSON, &it.City.Names); err != nil {
			return nil, fmt.Errorf("unmarshal favorite city names: %w", err)
		}

		// Set seller name (nullable)
		if sellerName != nil {
			it.Seller.Name = *sellerName
		}

		// Reconstruct price
		if priceAmount != nil && priceCurrency != nil {
			p, err := model.NewPrice(*priceAmount, strings.TrimSpace(*priceCurrency))
			if err != nil {
				return nil, fmt.Errorf("invalid stored price for favorite item %s: %w", it.ID, err)
			}
			it.Price = &p
		}

		itemIDs = append(itemIDs, it.ID)
		items = append(items, fav)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate favorites: %w", err)
	}

	// Fetch photos for all items in one query
	if len(itemIDs) > 0 {
		photoRows, err := r.db.Query(ctx,
			`SELECT id, item_id, uploader_id, bucket, object_key, mime_type, size_bytes, original_filename, sort_order, created_at
			 FROM item_photos WHERE item_id = ANY($1)
			 ORDER BY sort_order ASC, created_at ASC`,
			itemIDs,
		)
		if err != nil {
			return nil, fmt.Errorf("list favorite photos: %w", err)
		}
		defer photoRows.Close()

		photosByItem := make(map[uuid.UUID][]model.ItemPhoto)
		for photoRows.Next() {
			var p model.ItemPhoto
			if err := photoRows.Scan(
				&p.ID, &p.ItemID, &p.UploaderID, &p.Bucket, &p.ObjectKey, &p.MimeType,
				&p.SizeBytes, &p.OriginalFilename, &p.SortOrder, &p.CreatedAt,
			); err != nil {
				return nil, fmt.Errorf("scan favorite photo: %w", err)
			}
			photosByItem[*p.ItemID] = append(photosByItem[*p.ItemID], p)
		}
		if err := photoRows.Err(); err != nil {
			return nil, fmt.Errorf("iterate favorite photos: %w", err)
		}

		for i := range items {
			if photos, ok := photosByItem[items[i].Item.ID]; ok {
				items[i].Item.Photos = photos
			}
		}
	}

	meta := pp.Meta(total)
	return &ListResult{
		Items:      items,
		Total:      meta.Total,
		Page:       meta.Page,
		PerPage:    meta.PerPage,
		TotalPages: meta.TotalPages,
	}, nil
}

func (r *repository) Add(ctx context.Context, userID, itemID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_favorites (user_id, item_id) VALUES ($1, $2)`,
		userID, itemID,
	)
	return err
}

func (r *repository) Delete(ctx context.Context, userID, itemID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM user_favorites WHERE user_id = $1 AND item_id = $2`,
		userID, itemID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *repository) Exists(ctx context.Context, userID, itemID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_favorites WHERE user_id = $1 AND item_id = $2)`,
		userID, itemID,
	).Scan(&exists)
	return exists, err
}

func (r *repository) ExistsSet(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT item_id FROM user_favorites WHERE user_id = $1 AND item_id = ANY($2)`,
		userID, itemIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("exists set favorites: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan favorite id: %w", err)
		}
		result[id] = true
	}
	return result, rows.Err()
}

func IsFavoriteUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errorAsPg(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgErrCodeUniqueViolation && strings.Contains(pgErr.ConstraintName, "user_favorites")
}

func IsFavoriteItemFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errorAsPg(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgErrCodeForeignKeyViolation && pgErr.ConstraintName == "user_favorites_item_id_fkey"
}

func errorAsPg(err error, target **pgconn.PgError) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}

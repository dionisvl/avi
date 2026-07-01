package item

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
	"github.com/dionisvl/avi/api-go/internal/platform/locale"
	"github.com/dionisvl/avi/api-go/internal/platform/pagination"
)

type ListFilter struct {
	CategoryIDs []uuid.UUID
	CityID      *uuid.UUID
	Condition   string
	PriceMin    *int64
	PriceMax    *int64
	SellerID    *uuid.UUID
	Search      string
	Page        int
	PerPage     int
	OrderBy     string
	Statuses    []string // if empty defaults to ["published"]
}

type ListResult struct {
	Items      []model.ItemWithDetails
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

// ErrSlugConflict is returned by Create when the slug already exists (unique constraint violation).
var ErrSlugConflict = fmt.Errorf("slug conflict")

// IsSlugConflict reports whether err is a slug unique-constraint violation.
func IsSlugConflict(err error) bool {
	var pgErr *pgconn.PgError
	return err != nil && errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "slug")
}

type Repository interface {
	Create(ctx context.Context, i *model.Item) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.ItemWithDetails, error)
	GetBySlug(ctx context.Context, slug string) (*model.ItemWithDetails, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	List(ctx context.Context, f ListFilter) (*ListResult, error)
	Update(ctx context.Context, id uuid.UUID, fields map[string]any) (*model.ItemWithDetails, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

// splitPrice decomposes a Price into nullable DB columns (amount minor units, ISO-4217 code).
func splitPrice(p *model.Price) (*int64, *string) {
	if p == nil {
		return nil, nil
	}
	amount := p.Amount()
	currency := p.Currency().Code
	return &amount, &currency
}

func (r *repository) Create(ctx context.Context, i *model.Item) error {
	query := `
		INSERT INTO items
			(id, seller_id, created_by, slug, title, category_id, description, tags, city_id, condition, status, price_amount, price_currency, thumbnail_id, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	priceAmount, priceCurrency := splitPrice(i.Price)
	tags := []string{} // items.tags is NOT NULL; nil pointer would send NULL
	if i.Tags != nil {
		tags = *i.Tags
	}
	_, err := r.db.Exec(ctx, query,
		i.ID, i.SellerID, i.CreatedBy, i.Slug, i.Title, i.CategoryID, i.Description, tags, i.City.ID, i.Condition,
		i.Status, priceAmount, priceCurrency, i.ThumbnailID, i.CreatedAt, i.UpdatedAt,
	)
	if IsSlugConflict(err) {
		return ErrSlugConflict
	}
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*model.ItemWithDetails, error) {
	loc := locale.FromCtx(ctx)
	query := `
		SELECT
			i.id, i.seller_id, i.created_by, i.slug, i.title, i.category_id, i.description, i.tags, i.city_id, i.condition, i.status, i.price_amount, i.price_currency, i.thumbnail_id, i.created_at, i.updated_at,
			u.id, u.name,
			ic.id, ic.slug, ic.parent_id, ic.names, ic.sort_order, ic.is_active,
			icc.slug, icc.geoname_id, icc.names, icc.is_active, icc.population
		FROM items i
		JOIN users u ON u.id = i.seller_id
		JOIN categories ic ON ic.id = i.category_id
		JOIN cities icc ON icc.id = i.city_id
		WHERE i.id = $1
	`
	row := r.db.QueryRow(ctx, query, id)

	item, err := scanItemWithSeller(row, loc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	photos, err := r.getItemPhotos(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	item.Photos = photos

	if item.ThumbnailID != nil {
		thumbnail, err := r.getItemThumbnail(ctx, *item.ThumbnailID)
		if err != nil {
			return nil, err
		}
		item.Thumbnail = thumbnail
	}

	return item, nil
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*model.ItemWithDetails, error) {
	loc := locale.FromCtx(ctx)
	query := `
		SELECT
			i.id, i.seller_id, i.created_by, i.slug, i.title, i.category_id, i.description, i.tags, i.city_id, i.condition, i.status, i.price_amount, i.price_currency, i.thumbnail_id, i.created_at, i.updated_at,
			u.id, u.name,
			ic.id, ic.slug, ic.parent_id, ic.names, ic.sort_order, ic.is_active,
			icc.slug, icc.geoname_id, icc.names, icc.is_active, icc.population
		FROM items i
		JOIN users u ON u.id = i.seller_id
		JOIN categories ic ON ic.id = i.category_id
		JOIN cities icc ON icc.id = i.city_id
		WHERE i.slug = $1
	`
	row := r.db.QueryRow(ctx, query, slug)

	item, err := scanItemWithSeller(row, loc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	photos, err := r.getItemPhotos(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	item.Photos = photos

	if item.ThumbnailID != nil {
		thumbnail, err := r.getItemThumbnail(ctx, *item.ThumbnailID)
		if err != nil {
			return nil, err
		}
		item.Thumbnail = thumbnail
	}

	return item, nil
}

func (r *repository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

func (r *repository) List(ctx context.Context, f ListFilter) (*ListResult, error) {
	pp := pagination.NewParams(f.Page, f.PerPage)
	loc := locale.FromCtx(ctx)

	where, args := buildWhereClause(f)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM items i
		JOIN users u ON u.id = i.seller_id
		JOIN categories ic ON ic.id = i.category_id
		JOIN cities icc ON icc.id = i.city_id
		%s
	`, where)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count items: %w", err)
	}

	orderBy := f.OrderBy
	if orderBy == "" {
		orderBy = "i.created_at DESC"
	}

	args = append(args, pp.Limit(), pp.Offset())
	listQuery := fmt.Sprintf(`
		SELECT
			i.id, i.seller_id, i.created_by, i.slug, i.title, i.category_id, i.description, i.tags, i.city_id, i.condition, i.status, i.price_amount, i.price_currency, i.thumbnail_id, i.created_at, i.updated_at,
			u.id, u.name,
			ic.id, ic.slug, ic.parent_id, ic.names, ic.sort_order, ic.is_active,
			icc.slug, icc.geoname_id, icc.names, icc.is_active, icc.population
		FROM items i
		JOIN users u ON u.id = i.seller_id
		JOIN categories ic ON ic.id = i.category_id
		JOIN cities icc ON icc.id = i.city_id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	items := make([]model.ItemWithDetails, 0)
	for rows.Next() {
		it, err := scanItemWithSeller(rows, loc)
		if err != nil {
			return nil, err
		}
		items = append(items, *it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(items) > 0 {
		ids := make([]uuid.UUID, len(items))
		for i, it := range items {
			ids[i] = it.ID
		}
		photoMap, err := r.getItemPhotosBatch(ctx, ids)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].Photos = photoMap[items[i].ID]
		}

		thumbnailMap, err := r.getItemThumbnailsBatch(ctx, items)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].Thumbnail = thumbnailMap[items[i].ID]
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

func (r *repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) (*model.ItemWithDetails, error) {
	if len(fields) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields)+2)
	i := 1

	allowedFields := map[string]bool{
		"title": true, "category_id": true, "description": true, "tags": true,
		"city_id": true, "condition": true, "status": true,
		"price_amount": true, "price_currency": true, "thumbnail_id": true,
	}

	for col, val := range fields {
		if !allowedFields[col] {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, time.Now())
	i++
	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE items SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "),
		i,
	)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM items WHERE id = $1", id)
	return err
}

func (r *repository) getItemPhotos(ctx context.Context, itemID uuid.UUID) ([]model.ItemPhoto, error) {
	query := `
		SELECT id, item_id, uploader_id, bucket, object_key, mime_type, size_bytes, original_filename, sort_order, created_at
		FROM item_photos
		WHERE item_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`
	rows, err := r.db.Query(ctx, query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPhotos(rows)
}

func (r *repository) getItemPhotosBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]model.ItemPhoto, error) {
	query := `
		SELECT id, item_id, uploader_id, bucket, object_key, mime_type, size_bytes, original_filename, sort_order, created_at
		FROM item_photos
		WHERE item_id = ANY($1)
		ORDER BY sort_order ASC, created_at ASC
	`
	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]model.ItemPhoto)
	for rows.Next() {
		var p model.ItemPhoto
		if err := rows.Scan(&p.ID, &p.ItemID, &p.UploaderID, &p.Bucket, &p.ObjectKey, &p.MimeType, &p.SizeBytes, &p.OriginalFilename, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		if p.ItemID != nil {
			result[*p.ItemID] = append(result[*p.ItemID], p)
		}
	}
	return result, rows.Err()
}

func (r *repository) getItemThumbnail(ctx context.Context, id uuid.UUID) (*model.ItemPhoto, error) {
	query := `
		SELECT id, item_id, uploader_id, bucket, object_key, mime_type, size_bytes, original_filename, sort_order, created_at
		FROM item_photos
		WHERE id = $1
	`
	var p model.ItemPhoto
	if err := r.db.QueryRow(ctx, query, id).Scan(&p.ID, &p.ItemID, &p.UploaderID, &p.Bucket, &p.ObjectKey, &p.MimeType, &p.SizeBytes, &p.OriginalFilename, &p.SortOrder, &p.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *repository) getItemThumbnailsBatch(ctx context.Context, items []model.ItemWithDetails) (map[uuid.UUID]*model.ItemPhoto, error) {
	result := make(map[uuid.UUID]*model.ItemPhoto)

	thumbnailIDs := make([]uuid.UUID, 0)
	for _, it := range items {
		if it.ThumbnailID != nil {
			thumbnailIDs = append(thumbnailIDs, *it.ThumbnailID)
		}
	}

	if len(thumbnailIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT id, item_id, uploader_id, bucket, object_key, mime_type, size_bytes, original_filename, sort_order, created_at
		FROM item_photos
		WHERE id = ANY($1)
	`
	rows, err := r.db.Query(ctx, query, thumbnailIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	photoMap := make(map[uuid.UUID]*model.ItemPhoto)
	for rows.Next() {
		var p model.ItemPhoto
		if err := rows.Scan(&p.ID, &p.ItemID, &p.UploaderID, &p.Bucket, &p.ObjectKey, &p.MimeType, &p.SizeBytes, &p.OriginalFilename, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		photoMap[p.ID] = &p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, it := range items {
		if it.ThumbnailID != nil {
			result[it.ID] = photoMap[*it.ThumbnailID]
		}
	}

	return result, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanItemWithSeller(row scanner, loc string) (*model.ItemWithDetails, error) {
	it := &model.ItemWithDetails{}
	it.Category = &model.Category{} // scanned into below; Item.Category is a pointer
	var itemDesc *string
	var tags []string
	var categoryNamesJSON []byte
	var categoryParentID *uuid.UUID
	var cityNamesJSON []byte
	var sellerName *string
	var priceAmount *int64
	var priceCurrency *string

	err := row.Scan(
		&it.ID, &it.SellerID, &it.CreatedBy, &it.Slug, &it.Title, &it.CategoryID, &itemDesc, &tags, &it.City.ID, &it.Condition, &it.Status, &priceAmount, &priceCurrency, &it.ThumbnailID, &it.CreatedAt, &it.UpdatedAt,
		&it.Seller.ID, &sellerName,
		&it.Category.ID, &it.Category.Slug, &categoryParentID, &categoryNamesJSON, &it.Category.SortOrder, &it.Category.IsActive,
		&it.City.Slug, &it.City.GeonameID, &cityNamesJSON, &it.City.IsActive, &it.City.Population,
	)
	if err != nil {
		return nil, err
	}

	if itemDesc != nil {
		it.Description = *itemDesc
	}
	if len(tags) > 0 {
		it.Tags = &tags
	}

	// Unmarshal category names and resolve localized name
	if err := json.Unmarshal(categoryNamesJSON, &it.Category.Names); err != nil {
		return nil, fmt.Errorf("unmarshal item category names: %w", err)
	}
	it.Category.ParentID = categoryParentID
	it.Category.Name = it.Category.LocalizedName(loc)

	// Unmarshal city names
	if err := json.Unmarshal(cityNamesJSON, &it.City.Names); err != nil {
		return nil, fmt.Errorf("unmarshal item city names: %w", err)
	}

	// Set seller name (nullable)
	if sellerName != nil {
		it.Seller.Name = *sellerName
	}

	// Reconstruct price
	if priceAmount != nil && priceCurrency != nil {
		p, err := model.NewPrice(*priceAmount, strings.TrimSpace(*priceCurrency))
		if err != nil {
			return nil, fmt.Errorf("invalid stored price for item %s: %w", it.ID, err)
		}
		it.Price = &p
	}

	return it, nil
}

func scanPhotos(rows pgx.Rows) ([]model.ItemPhoto, error) {
	var photos []model.ItemPhoto
	for rows.Next() {
		var p model.ItemPhoto
		if err := rows.Scan(&p.ID, &p.ItemID, &p.UploaderID, &p.Bucket, &p.ObjectKey, &p.MimeType, &p.SizeBytes, &p.OriginalFilename, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

type whereBuilder struct {
	conds []string
	args  []any
}

// Add appends a condition with one or more positional placeholders.
// The cond string must contain exactly one "$%d" per value, in order; each is
// rendered to the next positional argument ($1, $2, ...).
func (w *whereBuilder) Add(cond string, vals ...any) {
	nums := make([]any, len(vals))
	for i := range vals {
		nums[i] = len(w.args) + 1 + i
	}
	w.conds = append(w.conds, fmt.Sprintf(cond, nums...))
	w.args = append(w.args, vals...)
}

func (w *whereBuilder) SQL() (string, []any) {
	if len(w.conds) == 0 {
		return "", w.args
	}
	return "WHERE " + strings.Join(w.conds, " AND "), w.args
}

// escapeLike escapes LIKE metacharacters (backslash, percent, underscore) so they match literally.
// Order matters: backslash must be escaped first to avoid re-escaping.
func escapeLike(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return r.Replace(s)
}

func buildWhereClause(f ListFilter) (string, []any) {
	w := &whereBuilder{}

	statuses := f.Statuses
	if len(statuses) == 0 {
		statuses = []string{"published"}
	}
	if len(statuses) == 1 {
		w.Add("i.status = $%d", statuses[0])
	} else {
		placeholders := make([]string, len(statuses))
		for i, s := range statuses {
			w.args = append(w.args, s)
			placeholders[i] = fmt.Sprintf("$%d::item_status", len(w.args))
		}
		w.conds = append(w.conds, "i.status IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(f.CategoryIDs) > 0 {
		w.args = append(w.args, f.CategoryIDs)
		w.conds = append(w.conds, fmt.Sprintf("i.category_id = ANY($%d)", len(w.args)))
	}

	if f.CityID != nil {
		w.Add("i.city_id = $%d", *f.CityID)
	}

	if f.Condition != "" {
		w.Add("i.condition = $%d", f.Condition)
	}

	if f.PriceMin != nil {
		w.Add("i.price_amount >= $%d", *f.PriceMin)
	}

	if f.PriceMax != nil {
		w.Add("i.price_amount <= $%d", *f.PriceMax)
	}

	if f.SellerID != nil {
		w.Add("i.seller_id = $%d", *f.SellerID)
	}

	if f.Search != "" {
		term := "%" + escapeLike(f.Search) + "%"
		w.Add("(i.title ILIKE $%d ESCAPE '\\' OR i.description ILIKE $%d ESCAPE '\\')", term, term)
	}

	return w.SQL()
}

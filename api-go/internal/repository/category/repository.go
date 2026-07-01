package category

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
)

type Repository interface {
	List(ctx context.Context, locale string) ([]model.Category, error)
	GetByID(ctx context.Context, id uuid.UUID, locale string) (*model.Category, error)
	GetBySlug(ctx context.Context, slug string, locale string) (*model.Category, error)
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) List(ctx context.Context, locale string) ([]model.Category, error) {
	query := `
		SELECT id, slug, parent_id, names, sort_order, is_active
		FROM categories
		WHERE is_active = TRUE
		ORDER BY sort_order ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	categories := make([]model.Category, 0)
	for rows.Next() {
		var c model.Category
		var namesJSON []byte
		if err := rows.Scan(&c.ID, &c.Slug, &c.ParentID, &namesJSON, &c.SortOrder, &c.IsActive); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(namesJSON, &c.Names); err != nil {
			return nil, fmt.Errorf("unmarshal category names: %w", err)
		}
		c.Name = c.LocalizedName(locale)
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID, locale string) (*model.Category, error) {
	query := `
		SELECT id, slug, parent_id, names, sort_order, is_active
		FROM categories
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)

	var c model.Category
	var namesJSON []byte
	if err := row.Scan(&c.ID, &c.Slug, &c.ParentID, &namesJSON, &c.SortOrder, &c.IsActive); err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("get category by id: %w", err)
	}
	if err := json.Unmarshal(namesJSON, &c.Names); err != nil {
		return nil, fmt.Errorf("unmarshal category names: %w", err)
	}
	c.Name = c.LocalizedName(locale)
	return &c, nil
}

func (r *repository) GetBySlug(ctx context.Context, slug string, locale string) (*model.Category, error) {
	query := `
		SELECT id, slug, parent_id, names, sort_order, is_active
		FROM categories
		WHERE slug = $1
	`
	row := r.db.QueryRow(ctx, query, slug)

	var c model.Category
	var namesJSON []byte
	if err := row.Scan(&c.ID, &c.Slug, &c.ParentID, &namesJSON, &c.SortOrder, &c.IsActive); err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("get category by slug: %w", err)
	}
	if err := json.Unmarshal(namesJSON, &c.Names); err != nil {
		return nil, fmt.Errorf("unmarshal category names: %w", err)
	}
	c.Name = c.LocalizedName(locale)
	return &c, nil
}

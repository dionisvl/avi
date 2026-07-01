package city

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
	List(ctx context.Context) ([]model.City, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.City, error)
	GetByGeonameID(ctx context.Context, geonameID int) (*model.City, error)
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) List(ctx context.Context) ([]model.City, error) {
	query := `SELECT id, slug, geoname_id, names, is_active, population FROM cities ORDER BY population DESC, slug ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list cities: %w", err)
	}
	defer rows.Close()

	cities := make([]model.City, 0)
	for rows.Next() {
		var c model.City
		var namesJSON []byte
		if err := rows.Scan(&c.ID, &c.Slug, &c.GeonameID, &namesJSON, &c.IsActive, &c.Population); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(namesJSON, &c.Names); err != nil {
			return nil, fmt.Errorf("unmarshal city names: %w", err)
		}
		cities = append(cities, c)
	}
	return cities, rows.Err()
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*model.City, error) {
	query := `SELECT id, slug, geoname_id, names, is_active, population FROM cities WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)

	var c model.City
	var namesJSON []byte
	if err := row.Scan(&c.ID, &c.Slug, &c.GeonameID, &namesJSON, &c.IsActive, &c.Population); err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("get city by id: %w", err)
	}
	if err := json.Unmarshal(namesJSON, &c.Names); err != nil {
		return nil, fmt.Errorf("unmarshal city names: %w", err)
	}
	return &c, nil
}

func (r *repository) GetByGeonameID(ctx context.Context, geonameID int) (*model.City, error) {
	query := `SELECT id, slug, geoname_id, names, is_active, population FROM cities WHERE geoname_id = $1`
	row := r.db.QueryRow(ctx, query, geonameID)

	var c model.City
	var namesJSON []byte
	if err := row.Scan(&c.ID, &c.Slug, &c.GeonameID, &namesJSON, &c.IsActive, &c.Population); err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("get city by geoname_id: %w", err)
	}
	if err := json.Unmarshal(namesJSON, &c.Names); err != nil {
		return nil, fmt.Errorf("unmarshal city names: %w", err)
	}
	return &c, nil
}

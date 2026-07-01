package cityresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/repository/city"
)

// ResolveCity returns the full City model by UUID or geoname_id.
func ResolveCity(ctx context.Context, cityUUID *uuid.UUID, geonameID *int, cityRepo city.Repository) (*model.City, error) {
	if cityUUID != nil && geonameID != nil {
		return nil, fmt.Errorf("cannot provide both city_uuid and geoname_id")
	}

	if cityUUID != nil {
		c, err := cityRepo.GetByID(ctx, *cityUUID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("unknown city_uuid: %s", cityUUID)
		}
		if err != nil {
			return nil, fmt.Errorf("resolve city by uuid: %w", err)
		}
		return c, nil
	}

	if geonameID != nil {
		c, err := cityRepo.GetByGeonameID(ctx, *geonameID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("unknown geoname_id: %d", *geonameID)
		}
		if err != nil {
			return nil, fmt.Errorf("resolve city by geoname_id: %w", err)
		}
		return c, nil
	}

	return nil, nil
}

// ResolveCityID is a convenience wrapper that returns only the city UUID.
func ResolveCityID(ctx context.Context, cityUUID *uuid.UUID, geonameID *int, cityRepo city.Repository) (*uuid.UUID, error) {
	c, err := ResolveCity(ctx, cityUUID, geonameID, cityRepo)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &c.ID, nil
}

package item

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ItemListQuery holds validated query parameters for GET /items.
// sort is intentionally excluded — it's handled separately via sort.Parse with a whitelist.
type ItemListQuery struct {
	CategoryID  *uuid.UUID  `query:"category_id"  validate:"omitempty,uuid"`
	CategoryIDs []uuid.UUID `query:"category_ids" validate:"omitempty,max=20,dive,uuid"`
	CityID      *uuid.UUID  `query:"city_uuid"    validate:"omitempty,uuid"`
	GeonameID   *int        `query:"geoname_id"   validate:"omitempty,min=1"`
	Condition   string      `query:"condition"    validate:"omitempty,oneof=new used"`
	PriceMin    *int64      `query:"price_min"    validate:"omitempty,min=0"`
	PriceMax    *int64      `query:"price_max"    validate:"omitempty,min=0"`
	SellerID    *uuid.UUID  `query:"seller_id"    validate:"omitempty,uuid"`
	Search      string      `query:"search"       validate:"omitempty,max=100"`
	Status      string      `query:"status"       validate:"omitempty,oneof=published draft archived sold"`
	// Statuses filters by one or more statuses (comma-separated, e.g. "published,archived").
	// Non-published statuses are only allowed when the authenticated viewer is the seller.
	Statuses []string `query:"statuses" validate:"omitempty,dive,oneof=published draft archived sold"`
}

// PriceRequest is the price payload for create/update: amount in minor units + ISO-4217 currency.
type PriceRequest struct {
	Amount   int64  `json:"amount"   validate:"min=0"`
	Currency string `json:"currency" validate:"required,len=3,alpha"`
}

// CreateItemRequest is the request body for POST /items
type CreateItemRequest struct {
	Title       string        `json:"title"        validate:"required,min=1,max=100"`
	CategoryID  uuid.UUID     `json:"category_id"  validate:"required,uuid"`
	Description string        `json:"description" validate:"omitempty,max=2000"`
	Condition   string        `json:"condition"   validate:"omitempty,oneof=new used"`
	Tags        *[]string     `json:"tags"        validate:"omitempty,max=20,dive,max=50"`
	PhotoIDs    []uuid.UUID   `json:"photo_ids"   validate:"omitempty,max=10"`
	ThumbnailID *uuid.UUID    `json:"thumbnail_id" validate:"omitempty,uuid"`
	CityID      *uuid.UUID    `json:"city_uuid"   validate:"omitempty,uuid"`
	GeonameID   *int          `json:"geoname_id"  validate:"omitempty,min=1"`
	Price       *PriceRequest `json:"price"       validate:"omitempty"`
}

// UpdateItemRequest is the request body for PATCH /items/{id}
type UpdateItemRequest struct {
	Title       *string      `json:"title"        validate:"omitempty,min=1,max=100"`
	CategoryID  *uuid.UUID   `json:"category_id"  validate:"omitempty,uuid"`
	Description *string      `json:"description" validate:"omitempty,max=2000"`
	Condition   *string      `json:"condition"   validate:"omitempty,oneof=new used"`
	Tags        *[]string    `json:"tags"        validate:"omitempty,max=20,dive,max=50"`
	PhotoIDs    *[]uuid.UUID `json:"photo_ids"   validate:"omitempty,max=10"`
	ThumbnailID *uuid.UUID   `json:"thumbnail_id" validate:"omitempty,uuid"`
	// ThumbnailIDSet distinguishes an omitted thumbnail_id from explicit null in PATCH JSON.
	ThumbnailIDSet bool          `json:"-"`
	CityID         *uuid.UUID    `json:"city_uuid"   validate:"omitempty,uuid"`
	GeonameID      *int          `json:"geoname_id"  validate:"omitempty,min=1"`
	Status         *string       `json:"status"      validate:"omitempty,oneof=published draft archived sold"`
	Price          *PriceRequest `json:"price"       validate:"omitempty"`
}

func (r *UpdateItemRequest) UnmarshalJSON(data []byte) error {
	type alias UpdateItemRequest
	aux := struct {
		*alias
		ThumbnailID json.RawMessage `json:"thumbnail_id"`
	}{
		alias: (*alias)(r),
	}

	r.ThumbnailID = nil
	r.ThumbnailIDSet = false

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ThumbnailID == nil {
		return nil
	}

	r.ThumbnailIDSet = true
	if bytes.Equal(aux.ThumbnailID, []byte("null")) {
		return nil
	}

	var id uuid.UUID
	if err := json.Unmarshal(aux.ThumbnailID, &id); err != nil {
		return fmt.Errorf("thumbnail_id: %w", err)
	}
	r.ThumbnailID = &id
	return nil
}

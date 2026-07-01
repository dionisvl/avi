package favoriteview

import (
	"time"

	"github.com/google/uuid"

	itemquery "github.com/dionisvl/avi/api-go/internal/query/item"
)

// ListFilter describes query parameters for favorite listing.
type ListFilter struct {
	Page    int
	PerPage int
}

// Pagination holds pagination metadata.
type Pagination struct {
	Page       int
	PerPage    int
	Total      int
	TotalPages int
}

// FavoriteItem holds favorite with item projection. No JSON tags — CQRS rule.
type FavoriteItem struct {
	ID        uuid.UUID
	ItemID    uuid.UUID
	CreatedAt time.Time
	Item      itemquery.Item
}

// ListResult holds the result of a list query.
type ListResult struct {
	Items      []FavoriteItem
	Pagination Pagination
}

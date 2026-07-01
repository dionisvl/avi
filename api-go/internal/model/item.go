package model

import (
	"errors"
	"strings"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/google/uuid"
)

const MaxItemPhotos = 10

// ErrInvalidCurrency is returned when a currency code is not a recognised ISO-4217 code.
var ErrInvalidCurrency = errors.New("invalid currency")

// Price is a multi-currency money value. Amount is stored in minor units
// (e.g. cents) and currency is an ISO-4217 code. It wraps go-money so callers
// get safe arithmetic, comparison and formatting.
type Price struct {
	*money.Money
}

// NewPrice builds a Price from minor units and an ISO-4217 currency code.
// It returns an error if the currency is not recognised.
func NewPrice(amountMinor int64, currencyCode string) (Price, error) {
	code := strings.ToUpper(strings.TrimSpace(currencyCode))
	cur := money.GetCurrency(code)
	if cur == nil {
		return Price{}, ErrInvalidCurrency
	}
	return Price{money.New(amountMinor, code)}, nil
}

type Seller struct {
	ID   uuid.UUID
	Name string
}

type Item struct {
	ID          uuid.UUID
	SellerID    uuid.UUID
	CreatedBy   *uuid.UUID // kept for audit trail; can equal SellerID
	Slug        string
	Title       string
	CategoryID  uuid.UUID
	Category    *Category
	Description string
	Tags        *[]string
	City        City
	Condition   ItemCondition
	Status      ItemStatus
	Price       *Price
	ThumbnailID *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ItemWithDetails struct {
	Item
	Seller    Seller
	Photos    []ItemPhoto
	Thumbnail *ItemPhoto
}

// NewItemInput holds validated data for constructing a new Item.
type NewItemInput struct {
	ID          uuid.UUID
	SellerID    uuid.UUID
	CreatedBy   *uuid.UUID
	Slug        string
	Title       string
	CategoryID  uuid.UUID
	Description string
	Tags        *[]string
	CityID      uuid.UUID
	Condition   ItemCondition
	Price       *Price
	ThumbnailID *uuid.UUID
}

// NewItem constructs an Item ready for persistence.
func NewItem(in NewItemInput) *Item {
	now := time.Now()
	return &Item{
		ID:          in.ID,
		SellerID:    in.SellerID,
		CreatedBy:   in.CreatedBy,
		Slug:        in.Slug,
		Title:       in.Title,
		CategoryID:  in.CategoryID,
		Description: in.Description,
		Tags:        in.Tags,
		City:        City{ID: in.CityID},
		Condition:   in.Condition,
		Price:       in.Price,
		ThumbnailID: in.ThumbnailID,
		Status:      ItemStatusPublished,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CanBeManagedBy returns true if userID is the seller or the creator of this item.
func (i *ItemWithDetails) CanBeManagedBy(userID uuid.UUID) bool {
	if i.SellerID == userID {
		return true
	}
	return i.CreatedBy != nil && *i.CreatedBy == userID
}

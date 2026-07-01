package item

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/media"
	itemrepo "github.com/dionisvl/avi/api-go/internal/repository/item"
)

// ListFilter describes query parameters for item listing.
type ListFilter struct {
	CategoryIDs      []uuid.UUID
	CityID           *uuid.UUID
	Condition        string
	PriceMin         *int64
	PriceMax         *int64
	SellerID         *uuid.UUID
	Search           string
	Page             int
	PerPage          int
	OrderBy          string // SQL ORDER BY fragment
	Statuses         []string
	RequireOwnership *uuid.UUID // if set, verifies SellerID == RequireOwnership for non-published items
}

type CityRef struct {
	ID         uuid.UUID
	Slug       string
	GeonameID  *int
	Names      map[string]string
	IsActive   bool
	Population int
}

type CategoryRef struct {
	ID   uuid.UUID
	Slug string
	Name string
}

type SellerRef struct {
	ID   uuid.UUID
	Name string
}

type PriceView struct {
	Amount   int64
	Currency string
}

type Photo struct {
	ID           uuid.UUID
	URL          string
	ThumbnailURL string
}

// Item is the read model for a classifieds listing. No JSON tags — CQRS rule.
type Item struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Description string
	CategoryID  uuid.UUID
	Category    *CategoryRef
	Condition   string
	Status      string
	Tags        *[]string
	Photos      []Photo
	Thumbnail   *Photo
	Seller      SellerRef
	City        CityRef
	Price       *PriceView
	CreatedAt   time.Time
	IsFavorited *bool
}

type Pagination struct {
	Page       int
	PerPage    int
	Total      int
	TotalPages int
}

type ListResult struct {
	Items      []Item
	Pagination Pagination
}

// ItemReader describes read-side capabilities for items.
type ItemReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.ItemWithDetails, error)
	GetBySlug(ctx context.Context, slug string) (*model.ItemWithDetails, error)
	List(ctx context.Context, f itemrepo.ListFilter) (*itemrepo.ListResult, error)
}

type FavoriteReader interface {
	Exists(ctx context.Context, userID, itemID uuid.UUID) (bool, error)
	ExistsSet(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type Service interface {
	List(ctx context.Context, f ListFilter, viewerID *uuid.UUID) (*ListResult, error)
	GetByID(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, isAdmin bool) (*Item, error)
	GetBySlug(ctx context.Context, slug string, viewerID *uuid.UUID, isAdmin bool) (*Item, error)
}

type service struct {
	items     ItemReader
	favorites FavoriteReader
	s3BaseURL string
}

func New(items ItemReader, favorites FavoriteReader, s3BaseURL string) Service {
	return &service{
		items:     items,
		favorites: favorites,
		s3BaseURL: s3BaseURL,
	}
}

func (s *service) List(ctx context.Context, f ListFilter, viewerID *uuid.UUID) (*ListResult, error) {
	// Verify RequireOwnership: if non-published items requested and ownership check is required,
	// ensure SellerID == RequireOwnership.
	if f.RequireOwnership != nil && f.SellerID != nil {
		if *f.SellerID != *f.RequireOwnership {
			return nil, apierr.New(apierr.ErrForbidden, "you can only view non-published items you own")
		}
	}

	// Adapt query filter to repo filter
	repoFilter := itemrepo.ListFilter{
		CategoryIDs: f.CategoryIDs,
		CityID:      f.CityID,
		Condition:   f.Condition,
		PriceMin:    f.PriceMin,
		PriceMax:    f.PriceMax,
		SellerID:    f.SellerID,
		Search:      f.Search,
		Page:        f.Page,
		PerPage:     f.PerPage,
		OrderBy:     f.OrderBy,
		Statuses:    f.Statuses,
	}

	result, err := s.items.List(ctx, repoFilter)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to list items")
	}

	favoritedSet, err := s.favoritedSet(ctx, viewerID, result.Items)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to fetch favorites")
	}

	items := make([]Item, 0, len(result.Items))
	for _, itemDetails := range result.Items {
		view := MapItem(itemDetails, s.s3BaseURL, isFavoritedPtr(viewerID, favoritedSet[itemDetails.ID]))
		items = append(items, view)
	}

	return &ListResult{
		Items: items,
		Pagination: Pagination{
			Page:       result.Page,
			PerPage:    result.PerPage,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	}, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, isAdmin bool) (*Item, error) {
	itemDetails, err := s.items.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierr.New(apierr.ErrNotFound, "item not found")
	}
	if err != nil {
		return nil, apierr.Wrap(err, "failed to fetch item")
	}

	// Check visibility: non-published items only visible to owner or admin
	isOwner := viewerID != nil && *viewerID == itemDetails.SellerID
	if !itemDetails.Status.IsPublished() && !isAdmin && !isOwner {
		return nil, apierr.New(apierr.ErrNotFound, "item not found")
	}

	isFavorited, err := s.isFavorited(ctx, viewerID, id)
	if err != nil {
		return nil, err
	}

	view := MapItem(*itemDetails, s.s3BaseURL, isFavorited)
	return &view, nil
}

func (s *service) GetBySlug(ctx context.Context, slug string, viewerID *uuid.UUID, isAdmin bool) (*Item, error) {
	itemDetails, err := s.items.GetBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierr.New(apierr.ErrNotFound, "item not found")
	}
	if err != nil {
		return nil, apierr.Wrap(err, "failed to fetch item")
	}

	// Check visibility: non-published items only visible to owner or admin
	isOwner := viewerID != nil && *viewerID == itemDetails.SellerID
	if !itemDetails.Status.IsPublished() && !isAdmin && !isOwner {
		return nil, apierr.New(apierr.ErrNotFound, "item not found")
	}

	isFavorited, err := s.isFavorited(ctx, viewerID, itemDetails.ID)
	if err != nil {
		return nil, err
	}

	view := MapItem(*itemDetails, s.s3BaseURL, isFavorited)
	return &view, nil
}

// MapItem builds an Item view from a model.ItemWithDetails.
func MapItem(it model.ItemWithDetails, s3BaseURL string, isFavorited *bool) Item {
	photos := make([]Photo, 0, len(it.Photos))
	for _, p := range it.Photos {
		url := media.URL(s3BaseURL, p.ObjectKey)
		photos = append(photos, Photo{
			ID:           p.ID,
			URL:          url,
			ThumbnailURL: url,
		})
	}

	var thumbnail *Photo
	if it.Thumbnail != nil {
		url := media.URL(s3BaseURL, it.Thumbnail.ObjectKey)
		thumbnail = &Photo{
			ID:           it.Thumbnail.ID,
			URL:          url,
			ThumbnailURL: url,
		}
	}

	var categoryRef *CategoryRef
	if it.Category != nil {
		categoryRef = &CategoryRef{
			ID:   it.Category.ID,
			Slug: it.Category.Slug,
			Name: it.Category.Name,
		}
	}

	cityRef := CityRef{
		ID:         it.City.ID,
		Slug:       it.City.Slug,
		GeonameID:  it.City.GeonameID,
		Names:      it.City.Names,
		IsActive:   it.City.IsActive,
		Population: it.City.Population,
	}

	var price *PriceView
	if it.Price != nil {
		price = &PriceView{
			Amount:   it.Price.Amount(),
			Currency: it.Price.Currency().Code,
		}
	}

	return Item{
		ID:          it.ID,
		Slug:        it.Slug,
		Title:       it.Title,
		Description: it.Description,
		CategoryID:  it.CategoryID,
		Category:    categoryRef,
		Condition:   string(it.Condition),
		Status:      string(it.Status),
		Tags:        it.Tags,
		Photos:      photos,
		Thumbnail:   thumbnail,
		Seller: SellerRef{
			ID:   it.Seller.ID,
			Name: it.Seller.Name,
		},
		City:        cityRef,
		Price:       price,
		CreatedAt:   it.CreatedAt,
		IsFavorited: isFavorited,
	}
}

func isFavoritedPtr(viewerID *uuid.UUID, isFavorited bool) *bool {
	if viewerID == nil {
		return nil
	}
	return &isFavorited
}

func (s *service) favoritedSet(ctx context.Context, viewerID *uuid.UUID, items []model.ItemWithDetails) (map[uuid.UUID]bool, error) {
	if viewerID == nil || s.favorites == nil || len(items) == 0 {
		return map[uuid.UUID]bool{}, nil
	}

	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return s.favorites.ExistsSet(ctx, *viewerID, ids)
}

func (s *service) isFavorited(ctx context.Context, viewerID *uuid.UUID, itemID uuid.UUID) (*bool, error) {
	if viewerID == nil || s.favorites == nil {
		return nil, nil
	}

	exists, err := s.favorites.Exists(ctx, *viewerID, itemID)
	if err != nil {
		return nil, err
	}

	return isFavoritedPtr(viewerID, exists), nil
}

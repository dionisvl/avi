package item

import (
	"time"

	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	"github.com/dionisvl/avi/api-go/internal/model"
	itemquery "github.com/dionisvl/avi/api-go/internal/query/item"
)

type PhotoResponse struct {
	ID           uuid.UUID `json:"id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
}

type CityResponse struct {
	ID         uuid.UUID         `json:"id"`
	Slug       string            `json:"slug"`
	GeonameID  *int              `json:"geoname_id"`
	Names      map[string]string `json:"names"`
	IsActive   bool              `json:"is_active"`
	Population int               `json:"population"`
}

type CategoryResponse struct {
	ID   uuid.UUID `json:"id"`
	Slug string    `json:"slug"`
	Name string    `json:"name"`
}

type SellerResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type PriceResponse struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type ItemResponse struct {
	ID          uuid.UUID         `json:"id"`
	Slug        string            `json:"slug"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	CategoryID  uuid.UUID         `json:"category_id"`
	Condition   string            `json:"condition"`
	Status      string            `json:"status"`
	Category    *CategoryResponse `json:"category,omitempty"`
	City        CityResponse      `json:"city"`
	Seller      SellerResponse    `json:"seller"`
	Tags        []string          `json:"tags"`
	Photos      []PhotoResponse   `json:"photos"`
	Thumbnail   *PhotoResponse    `json:"thumbnail,omitempty"`
	Price       *PriceResponse    `json:"price,omitempty"`
	IsFavorited *bool             `json:"is_favorited,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type ItemListResponse struct {
	Data       []ItemResponse         `json:"data"`
	Pagination api.PaginationResponse `json:"pagination"`
}

type ItemSingleResponse struct {
	Data ItemResponse `json:"data"`
}

func MapCityToResponse(c model.City) CityResponse {
	return CityResponse{
		ID:         c.ID,
		Slug:       c.Slug,
		GeonameID:  c.GeonameID,
		Names:      c.Names,
		IsActive:   c.IsActive,
		Population: c.Population,
	}
}

func MapCityPtrToResponse(c *model.City) *CityResponse {
	if c == nil {
		return nil
	}
	r := MapCityToResponse(*c)
	return &r
}

func MapItemToResponse(it itemquery.Item) ItemResponse {
	photos := make([]PhotoResponse, 0, len(it.Photos))
	for _, p := range it.Photos {
		photos = append(photos, PhotoResponse{
			ID:           p.ID,
			URL:          p.URL,
			ThumbnailURL: p.ThumbnailURL,
		})
	}

	tags := make([]string, 0)
	if it.Tags != nil {
		tags = *it.Tags
	}

	var categoryResp *CategoryResponse
	if it.Category != nil {
		categoryResp = &CategoryResponse{
			ID:   it.Category.ID,
			Slug: it.Category.Slug,
			Name: it.Category.Name,
		}
	}

	cityResp := CityResponse{
		ID:         it.City.ID,
		Slug:       it.City.Slug,
		GeonameID:  it.City.GeonameID,
		Names:      it.City.Names,
		IsActive:   it.City.IsActive,
		Population: it.City.Population,
	}

	var thumbnail *PhotoResponse
	if it.Thumbnail != nil {
		thumbnail = &PhotoResponse{
			ID:           it.Thumbnail.ID,
			URL:          it.Thumbnail.URL,
			ThumbnailURL: it.Thumbnail.ThumbnailURL,
		}
	}

	return ItemResponse{
		ID:          it.ID,
		Slug:        it.Slug,
		Title:       it.Title,
		Description: it.Description,
		CategoryID:  it.CategoryID,
		Condition:   it.Condition,
		Status:      it.Status,
		Category:    categoryResp,
		City:        cityResp,
		Seller:      SellerResponse{ID: it.Seller.ID, Name: it.Seller.Name},
		Tags:        tags,
		Photos:      photos,
		Thumbnail:   thumbnail,
		Price:       MapPriceToResponse(it.Price),
		IsFavorited: it.IsFavorited,
		CreatedAt:   it.CreatedAt,
	}
}

func MapPriceToResponse(p *itemquery.PriceView) *PriceResponse {
	if p == nil {
		return nil
	}
	return &PriceResponse{Amount: p.Amount, Currency: p.Currency}
}

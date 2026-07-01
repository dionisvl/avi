package item

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
	"github.com/dionisvl/avi/api-go/internal/platform/locale"
	"github.com/dionisvl/avi/api-go/internal/platform/slug"
	categoryrepo "github.com/dionisvl/avi/api-go/internal/repository/category"
	itemrepo "github.com/dionisvl/avi/api-go/internal/repository/item"
	mediarepo "github.com/dionisvl/avi/api-go/internal/repository/media"
)

// PriceInput carries a raw price from the API layer: amount in minor units + ISO-4217 currency.
type PriceInput struct {
	Amount   int64
	Currency string
}

type CreateInput struct {
	Title       string
	CategoryID  uuid.UUID
	Description string
	Condition   string
	Tags        *[]string
	PhotoIDs    []uuid.UUID
	ThumbnailID *uuid.UUID
	CityID      uuid.UUID
	CitySlug    string
	CreatedBy   uuid.UUID // the authenticated user making the request; they are the seller
	IsAdmin     bool
	Price       *PriceInput
}

type UpdateInput struct {
	Title          *string
	CategoryID     *uuid.UUID
	Description    *string
	Condition      *string
	Tags           *[]string
	PhotoIDs       *[]uuid.UUID
	ThumbnailID    *uuid.UUID
	ThumbnailIDSet bool
	CityID         *uuid.UUID
	Status         *string
	Price          *PriceInput
	RequestedBy    uuid.UUID // the authenticated user making the request
	IsAdmin        bool
}

type DeleteInput struct {
	ID          uuid.UUID
	RequestedBy uuid.UUID // the authenticated user making the request
	IsAdmin     bool
}

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
	OrderBy          string     // SQL ORDER BY fragment passed through from handler
	Statuses         []string   // if empty defaults to ["published"]
	RequireOwnership *uuid.UUID // if set, verifies that the requested SellerID == RequireOwnership
}

type Service interface {
	Create(ctx context.Context, in CreateInput) (*model.ItemWithDetails, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.ItemWithDetails, error)
	GetBySlug(ctx context.Context, slug string) (*model.ItemWithDetails, error)
	List(ctx context.Context, f ListFilter) (*itemrepo.ListResult, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*model.ItemWithDetails, error)
	Delete(ctx context.Context, in DeleteInput) error
	// CanManage returns true if userID is the seller of the item, or if isAdmin is true.
	CanManage(ctx context.Context, itemID uuid.UUID, userID uuid.UUID, isAdmin bool) (bool, error)
}

type service struct {
	repo         itemrepo.Repository
	categoryRepo categoryrepo.Repository
	mediaRepo    mediarepo.Repository
	db           dbtx.TxBeginner
	logger       *slog.Logger
}

func New(repo itemrepo.Repository, categoryRepo categoryrepo.Repository, mediaRepo mediarepo.Repository, db dbtx.TxBeginner, logger *slog.Logger) Service {
	return &service{repo: repo, categoryRepo: categoryRepo, mediaRepo: mediaRepo, db: db, logger: logger}
}

func (s *service) Create(ctx context.Context, in CreateInput) (*model.ItemWithDetails, error) {
	condition := model.ItemConditionUsed
	if in.Condition != "" {
		c, err := model.NewItemCondition(in.Condition)
		if err != nil {
			return nil, mapDomainError(err)
		}
		condition = c
	}

	if len(in.PhotoIDs) > model.MaxItemPhotos {
		return nil, apierr.New(apierr.ErrValidation, fmt.Sprintf("photo_ids must have at most %d items", model.MaxItemPhotos))
	}

	price, err := buildPrice(in.Price)
	if err != nil {
		return nil, err
	}

	if err := s.validateCategory(ctx, in.CategoryID); err != nil {
		return nil, err
	}

	if err := s.validatePhotoOwnership(ctx, in.PhotoIDs, in.CreatedBy, in.IsAdmin); err != nil {
		return nil, err
	}

	if in.ThumbnailID != nil {
		if err := s.validatePhotoOwnership(ctx, []uuid.UUID{*in.ThumbnailID}, in.CreatedBy, in.IsAdmin); err != nil {
			return nil, err
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, apierr.Wrap(err, "failed to generate id")
	}

	itemSlug, err := s.generateUniqueSlug(ctx, in.Title, in.CitySlug, id)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to generate slug")
	}

	itm := model.NewItem(model.NewItemInput{
		ID:          id,
		SellerID:    in.CreatedBy,
		CreatedBy:   &in.CreatedBy,
		Slug:        itemSlug,
		Title:       in.Title,
		CategoryID:  in.CategoryID,
		Description: in.Description,
		Tags:        in.Tags,
		CityID:      in.CityID,
		Condition:   condition,
		Price:       price,
		ThumbnailID: in.ThumbnailID,
	})

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to start transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txItemRepo := itemrepo.New(tx)
	if err := txItemRepo.Create(ctx, itm); err != nil {
		if itemrepo.IsSlugConflict(err) {
			// race: another request inserted the same base slug between our check and insert — use UUID suffix
			itm.Slug = itm.Slug + "-" + id.String()
			if err2 := txItemRepo.Create(ctx, itm); err2 != nil {
				return nil, apierr.Wrap(err2, "failed to create item")
			}
		} else {
			return nil, apierr.Wrap(err, "failed to create item")
		}
	}

	if err := s.linkPhotosWithRepo(ctx, mediarepo.New(tx), id, in.PhotoIDs); err != nil {
		s.logger.Error("failed to attach photos", slog.String("item_id", id.String()), slog.String("error", err.Error()))
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apierr.Wrap(err, "failed to commit transaction")
	}

	result, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to fetch created item")
	}

	return result, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*model.ItemWithDetails, error) {
	item, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierr.New(apierr.ErrNotFound, fmt.Sprintf("item %s not found", id))
	}
	if err != nil {
		return nil, apierr.Wrap(err, "failed to fetch item")
	}
	return item, nil
}

func (s *service) GetBySlug(ctx context.Context, sl string) (*model.ItemWithDetails, error) {
	item, err := s.repo.GetBySlug(ctx, sl)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierr.New(apierr.ErrNotFound, fmt.Sprintf("item with slug %q not found", sl))
	}
	if err != nil {
		return nil, apierr.Wrap(err, "failed to fetch item by slug")
	}
	return item, nil
}

func (s *service) List(ctx context.Context, f ListFilter) (*itemrepo.ListResult, error) {
	if f.RequireOwnership != nil && f.SellerID != nil {
		if *f.SellerID != *f.RequireOwnership {
			return nil, apierr.New(apierr.ErrForbidden, "you can only view non-published items for your own seller profile")
		}
	}

	result, err := s.repo.List(ctx, itemrepo.ListFilter{
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
	})
	if err != nil {
		return nil, apierr.Wrap(err, "failed to list items")
	}
	return result, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*model.ItemWithDetails, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !in.IsAdmin && !existing.CanBeManagedBy(in.RequestedBy) {
		return nil, apierr.New(apierr.ErrForbidden, "you can only edit items you own or created")
	}

	fields := map[string]any{}
	if in.Title != nil {
		fields["title"] = *in.Title
	}

	if in.CategoryID != nil {
		if err := s.validateCategory(ctx, *in.CategoryID); err != nil {
			return nil, err
		}
		fields["category_id"] = *in.CategoryID
	}

	if in.Description != nil {
		fields["description"] = *in.Description
	}

	if in.Condition != nil {
		c, err := model.NewItemCondition(*in.Condition)
		if err != nil {
			return nil, mapDomainError(err)
		}
		fields["condition"] = c
	}

	if in.Tags != nil {
		fields["tags"] = in.Tags
	}

	if in.CityID != nil {
		fields["city_id"] = *in.CityID
	}

	if in.Status != nil {
		next := model.ItemStatus(*in.Status)
		if !existing.Status.CanTransitionTo(next) {
			return nil, apierr.New(apierr.ErrBadRequest, "invalid status transition")
		}
		fields["status"] = next
	}

	if in.Price != nil {
		price, err := buildPrice(in.Price)
		if err != nil {
			return nil, err
		}
		amount, currency := price.Amount(), price.Currency().Code
		fields["price_amount"] = amount
		fields["price_currency"] = currency
	}

	if in.PhotoIDs != nil && len(*in.PhotoIDs) > model.MaxItemPhotos {
		return nil, apierr.New(apierr.ErrValidation, fmt.Sprintf("photo_ids must have at most %d items", model.MaxItemPhotos))
	}

	if in.PhotoIDs != nil && len(*in.PhotoIDs) > 0 {
		if err := s.validatePhotoOwnership(ctx, *in.PhotoIDs, in.RequestedBy, in.IsAdmin); err != nil {
			return nil, err
		}
	}

	if in.ThumbnailIDSet {
		if in.ThumbnailID != nil {
			if err := s.validatePhotoOwnership(ctx, []uuid.UUID{*in.ThumbnailID}, in.RequestedBy, in.IsAdmin); err != nil {
				return nil, err
			}
			fields["thumbnail_id"] = in.ThumbnailID
		} else {
			fields["thumbnail_id"] = nil
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to start transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if in.PhotoIDs != nil {
		if err := mediarepo.New(tx).ReplaceItemPhotos(ctx, id, *in.PhotoIDs); err != nil {
			s.logger.Error("failed to replace photos on update", slog.String("item_id", id.String()), slog.String("error", err.Error()))
			return nil, apierr.Wrap(err, "failed to replace item photos")
		}
	}

	result, err := itemrepo.New(tx).Update(ctx, id, fields)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to update item")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apierr.Wrap(err, "failed to commit transaction")
	}

	return result, nil
}

func (s *service) Delete(ctx context.Context, in DeleteInput) error {
	existing, err := s.GetByID(ctx, in.ID)
	if err != nil {
		return err
	}

	if !in.IsAdmin && !existing.CanBeManagedBy(in.RequestedBy) {
		return apierr.New(apierr.ErrForbidden, "you can only delete items you own or created")
	}

	if err := s.repo.Delete(ctx, in.ID); err != nil {
		return apierr.Wrap(err, "failed to delete item")
	}
	return nil
}

// validatePhotoOwnership checks that all photoIDs were uploaded by uploaderID (admin bypasses).
// Call this BEFORE creating the item to avoid orphaned records on ownership failure.
func (s *service) validatePhotoOwnership(ctx context.Context, photoIDs []uuid.UUID, uploaderID uuid.UUID, isAdmin bool) error {
	if len(photoIDs) == 0 || isAdmin {
		return nil
	}
	uploaderMap, err := s.mediaRepo.GetItemPhotoUploaderIDs(ctx, photoIDs)
	if err != nil {
		return apierr.Wrap(err, "failed to check photo ownership")
	}
	for _, photoID := range photoIDs {
		uid, found := uploaderMap[photoID]
		if !found || uid == nil || *uid != uploaderID {
			return apierr.New(apierr.ErrForbidden, "one or more photos not found or access denied")
		}
	}
	return nil
}

// linkPhotosWithRepo associates already-validated photoIDs with an item using the provided repo (e.g. a tx-scoped one).
func (s *service) linkPhotosWithRepo(ctx context.Context, repo mediarepo.Repository, itemID uuid.UUID, photoIDs []uuid.UUID) error {
	for _, photoID := range photoIDs {
		if err := repo.SetItemPhotoItemID(ctx, photoID, itemID); err != nil {
			return fmt.Errorf("attach photo %s: %w", photoID, err)
		}
	}
	return nil
}

func (s *service) CanManage(ctx context.Context, itemID uuid.UUID, userID uuid.UUID, isAdmin bool) (bool, error) {
	if isAdmin {
		return true, nil
	}
	item, err := s.repo.GetByID(ctx, itemID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, apierr.Wrap(err, "failed to fetch item")
	}
	return item.CanBeManagedBy(userID), nil
}

// buildPrice converts an optional PriceInput into a domain Price.
// Returns (nil, nil) when no price is provided.
func buildPrice(in *PriceInput) (*model.Price, error) {
	if in == nil {
		return nil, nil
	}
	if in.Amount < 0 {
		return nil, apierr.New(apierr.ErrValidation, "price amount must be non-negative")
	}
	p, err := model.NewPrice(in.Amount, in.Currency)
	if err != nil {
		return nil, apierr.New(apierr.ErrValidation, "price currency must be a valid ISO-4217 code")
	}
	return &p, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, model.ErrInvalidItemCondition):
		return apierr.New(apierr.ErrValidation, "condition must be 'new' or 'used'")
	default:
		return err
	}
}

// validateCategory checks that the category exists.
func (s *service) validateCategory(ctx context.Context, categoryID uuid.UUID) error {
	_, err := s.categoryRepo.GetByID(ctx, categoryID, locale.Default)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(apierr.ErrValidation, "category not found")
	}
	if err != nil {
		return apierr.Wrap(err, "failed to validate category")
	}
	return nil
}

// generateUniqueSlug builds slug from title+citySlug; on collision falls back to full UUID suffix.
func (s *service) generateUniqueSlug(ctx context.Context, title, citySlug string, id uuid.UUID) (string, error) {
	base := slug.Generate(title, citySlug)
	if base == "" {
		base = slug.Generate(id.String())
	}

	exists, err := s.repo.SlugExists(ctx, base)
	if err != nil {
		return "", err
	}
	if !exists {
		return base, nil
	}

	// collision: use full UUID to guarantee uniqueness
	return base + "-" + id.String(), nil
}

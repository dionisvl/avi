package favorite

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	favrepo "github.com/dionisvl/avi/api-go/internal/repository/favorite"
)

type ListResult struct {
	Items      []model.FavoriteWithItem
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

type Service interface {
	List(ctx context.Context, userID uuid.UUID, page, perPage int) (*ListResult, error)
	Add(ctx context.Context, userID, itemID uuid.UUID) error
	Remove(ctx context.Context, userID, itemID uuid.UUID) error
	Exists(ctx context.Context, userID, itemID uuid.UUID) (bool, error)
	ExistsSet(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type service struct {
	repo   favrepo.Repository
	logger *slog.Logger
}

func New(repo favrepo.Repository, logger *slog.Logger) Service {
	return &service{repo: repo, logger: logger}
}

func (s *service) List(ctx context.Context, userID uuid.UUID, page, perPage int) (*ListResult, error) {
	res, err := s.repo.List(ctx, userID, page, perPage)
	if err != nil {
		s.logger.Error("list favorites", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to list favorites")
	}
	return &ListResult{
		Items:      res.Items,
		Total:      res.Total,
		Page:       res.Page,
		PerPage:    res.PerPage,
		TotalPages: res.TotalPages,
	}, nil
}

func (s *service) Add(ctx context.Context, userID, itemID uuid.UUID) error {
	if err := s.repo.Add(ctx, userID, itemID); err != nil {
		if favrepo.IsFavoriteUniqueViolation(err) {
			return apierr.New(apierr.ErrAlreadyExists, "Item is already in favorites")
		}
		if favrepo.IsFavoriteItemFKViolation(err) {
			return apierr.New(apierr.ErrNotFound, "Item not found")
		}
		s.logger.Error("add favorite", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to add favorite")
	}
	return nil
}

func (s *service) Remove(ctx context.Context, userID, itemID uuid.UUID) error {
	if err := s.repo.Delete(ctx, userID, itemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(apierr.ErrNotFound, "Favorite not found")
		}
		s.logger.Error("remove favorite", slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to remove favorite")
	}
	return nil
}

func (s *service) Exists(ctx context.Context, userID, itemID uuid.UUID) (bool, error) {
	exists, err := s.repo.Exists(ctx, userID, itemID)
	if err != nil {
		s.logger.Error("favorite exists", slog.String("error", err.Error()))
		return false, apierr.New(apierr.ErrInternal, "Failed to resolve favorite state")
	}
	return exists, nil
}

func (s *service) ExistsSet(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result, err := s.repo.ExistsSet(ctx, userID, itemIDs)
	if err != nil {
		s.logger.Error("favorite exists set", slog.String("error", err.Error()))
		return nil, apierr.New(apierr.ErrInternal, "Failed to resolve favorite states")
	}
	return result, nil
}

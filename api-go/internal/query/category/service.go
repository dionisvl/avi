package category

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	catrepo "github.com/dionisvl/avi/api-go/internal/repository/category"
)

type Service interface {
	List(ctx context.Context, locale string) ([]model.Category, error)
	GetByID(ctx context.Context, id uuid.UUID, locale string) (*model.Category, error)
}

type service struct {
	repo catrepo.Repository
}

func New(repo catrepo.Repository) Service {
	return &service{repo: repo}
}

func (s *service) List(ctx context.Context, locale string) ([]model.Category, error) {
	categories, err := s.repo.List(ctx, locale)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to list categories")
	}
	return categories, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID, locale string) (*model.Category, error) {
	c, err := s.repo.GetByID(ctx, id, locale)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apierr.New(apierr.ErrNotFound, "category not found")
		}
		return nil, apierr.Wrap(err, "failed to get category")
	}
	return c, nil
}

package favoriteview

import (
	"context"

	"github.com/google/uuid"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	itemquery "github.com/dionisvl/avi/api-go/internal/query/item"
	favrepo "github.com/dionisvl/avi/api-go/internal/repository/favorite"
)

// Service describes favorite read operations.
type Service interface {
	List(ctx context.Context, userID uuid.UUID, filter ListFilter) (*ListResult, error)
}

type service struct {
	repo      favrepo.Repository
	s3BaseURL string
}

// New creates a new favorite view service.
func New(repo favrepo.Repository, s3BaseURL string) Service {
	return &service{repo: repo, s3BaseURL: s3BaseURL}
}

func (s *service) List(ctx context.Context, userID uuid.UUID, filter ListFilter) (*ListResult, error) {
	result, err := s.repo.List(ctx, userID, filter.Page, filter.PerPage)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to list favorites")
	}

	items := make([]FavoriteItem, 0, len(result.Items))
	for _, f := range result.Items {
		favorited := true
		itemView := itemquery.MapItem(f.Item, s.s3BaseURL, &favorited)
		items = append(items, FavoriteItem{
			ID:        f.ID,
			ItemID:    f.ItemID,
			CreatedAt: f.CreatedAt,
			Item:      itemView,
		})
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

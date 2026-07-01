package favorite

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	itemapi "github.com/dionisvl/avi/api-go/internal/api/item"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/platform/pagination"
	favoriteview "github.com/dionisvl/avi/api-go/internal/query/favoriteview"
	favoriteservice "github.com/dionisvl/avi/api-go/internal/service/favorite"
)

type Handler struct {
	writeSvc favoriteservice.Service
	readSvc  favoriteview.Service
	logger   *slog.Logger
}

func NewHandler(writeSvc favoriteservice.Service, readSvc favoriteview.Service, logger *slog.Logger) *Handler {
	return &Handler{writeSvc: writeSvc, readSvc: readSvc, logger: logger}
}

func (h *Handler) Routes(authSvc apimiddleware.TokenValidator) chi.Router {
	r := chi.NewRouter()
	r.Use(apimiddleware.AuthRequired(authSvc))

	r.Get("/", h.list)
	r.Post("/", h.add)
	r.Delete("/{item_id}", h.remove)

	return r
}

type AddRequest struct {
	ItemID uuid.UUID `json:"item_id" validate:"required"`
}

type FavoriteItemResponse struct {
	ID        uuid.UUID            `json:"id"`
	ItemID    uuid.UUID            `json:"item_id"`
	CreatedAt time.Time            `json:"created_at"`
	Item      itemapi.ItemResponse `json:"item"`
}

type FavoritesListResponse struct {
	Data       []FavoriteItemResponse `json:"data"`
	Pagination api.PaginationResponse `json:"pagination"`
}

func mapFavoriteViewToResponse(f favoriteview.FavoriteItem) FavoriteItemResponse {
	return FavoriteItemResponse{
		ID:        f.ID,
		ItemID:    f.ItemID,
		CreatedAt: f.CreatedAt,
		Item:      itemapi.MapItemToResponse(f.Item),
	}
}

// @Summary List favorites
// @Tags favorites
// @Produce json
// @Security BearerAuth
// @Param page     query int false "Page number (default: 1)"
// @Param per_page query int false "Items per page (default: 20)"
// @Success 200 {object} FavoritesListResponse
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/items/favorites [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	pp, err := pagination.ParseParams(r)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, err.Error()))
		return
	}

	result, err := h.readSvc.List(r.Context(), userID, favoriteview.ListFilter{Page: pp.Page, PerPage: pp.PerPage})
	if err != nil {
		api.Error(w, h.logger, api.AsAppError(err))
		return
	}

	items := make([]FavoriteItemResponse, 0, len(result.Items))
	for _, f := range result.Items {
		items = append(items, mapFavoriteViewToResponse(f))
	}

	api.JSON(w, http.StatusOK, FavoritesListResponse{
		Data: items,
		Pagination: api.PaginationResponse{
			Page:       result.Pagination.Page,
			PerPage:    result.Pagination.PerPage,
			Total:      result.Pagination.Total,
			TotalPages: result.Pagination.TotalPages,
		},
	})
}

// @Summary Add item to favorites
// @Tags favorites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body AddRequest true "Item ID"
// @Success 201 {object} map[string]string
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Failure 404 {object} api.ProblemDetails
// @Failure 409 {object} api.ProblemDetails
// @Router /api/v1/items/favorites [post]
func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	req, err := api.DecodeAndValidate[AddRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.writeSvc.Add(r.Context(), userID, req.ItemID); err != nil {
		api.Error(w, h.logger, api.AsAppError(err))
		return
	}

	api.JSON(w, http.StatusCreated, map[string]string{"message": "Added to favorites"})
}

// @Summary Remove item from favorites
// @Tags favorites
// @Produce json
// @Security BearerAuth
// @Param item_id path string true "Item ID"
// @Success 204
// @Failure 401 {object} api.ProblemDetails
// @Failure 404 {object} api.ProblemDetails
// @Router /api/v1/items/favorites/{item_id} [delete]
func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	itemIDStr := chi.URLParam(r, "item_id")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "Invalid item_id"))
		return
	}

	if err := h.writeSvc.Remove(r.Context(), userID, itemID); err != nil {
		api.Error(w, h.logger, api.AsAppError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

package categories

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/locale"
	"github.com/dionisvl/avi/api-go/internal/query/category"
)

type Handler struct {
	svc    category.Service
	logger *slog.Logger
}

func NewHandler(svc category.Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/{id}", h.getByID)
	return r
}

type CategoryResponse struct {
	ID        uuid.UUID         `json:"id"`
	Slug      string            `json:"slug"`
	ParentID  *uuid.UUID        `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	Names     map[string]string `json:"names"`
	SortOrder int16             `json:"sort_order"`
}

type ListResponse struct {
	Data []CategoryResponse `json:"data"`
}

type SingleResponse struct {
	Data CategoryResponse `json:"data"`
}

func mapCategory(c model.Category) CategoryResponse {
	return CategoryResponse{
		ID:        c.ID,
		Slug:      c.Slug,
		ParentID:  c.ParentID,
		Name:      c.Name,
		Names:     c.Names,
		SortOrder: c.SortOrder,
	}
}

// @Summary List categories
// @Tags categories
// @Produce json
// @Param locale  query string false "Locale (en or ru, default en)" Enums(en, ru)
// @Success 200 {object} ListResponse
// @Failure 400 {object} api.ProblemDetails
// @Router /api/v1/categories [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	loc := locale.FromCtx(r.Context())

	categories, err := h.svc.List(r.Context(), loc)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	resp := make([]CategoryResponse, len(categories))
	for i, c := range categories {
		resp[i] = mapCategory(c)
	}
	api.JSON(w, http.StatusOK, ListResponse{Data: resp})
}

// @Summary Get category by ID
// @Tags categories
// @Produce json
// @Param id     path  string true  "Category UUID"
// @Param locale query string false "Locale (en or ru, default en)" Enums(en, ru)
// @Success 200 {object} SingleResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 404 {object} api.ProblemDetails
// @Router /api/v1/categories/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "invalid category id"))
		return
	}

	loc := locale.FromCtx(r.Context())

	result, svcErr := h.svc.GetByID(r.Context(), id, loc)
	if svcErr != nil {
		api.Error(w, h.logger, svcErr)
		return
	}

	resp := mapCategory(*result)
	api.JSON(w, http.StatusOK, SingleResponse{Data: resp})
}

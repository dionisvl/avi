package item

import (
	"log/slog"
	"net/http"
	"regexp"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	"github.com/dionisvl/avi/api-go/internal/api/cityresolver"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/pagination"
	itemsort "github.com/dionisvl/avi/api-go/internal/platform/sort"
	itemquery "github.com/dionisvl/avi/api-go/internal/query/item"
	cityrepo "github.com/dionisvl/avi/api-go/internal/repository/city"
	itemservice "github.com/dionisvl/avi/api-go/internal/service/item"
)

// reSlug matches valid slugs: lowercase alphanum + dashes, max 197 chars (160 base + "-" + 36 uuid).
var reSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,195}[a-z0-9]$`)

type Handler struct {
	writeSvc itemservice.Service
	readSvc  itemquery.Service
	cityRepo cityrepo.Repository
	logger   *slog.Logger
}

func NewHandler(writeSvc itemservice.Service, readSvc itemquery.Service, cityRepo cityrepo.Repository, logger *slog.Logger) *Handler {
	return &Handler{writeSvc: writeSvc, readSvc: readSvc, cityRepo: cityRepo, logger: logger}
}

func (h *Handler) Routes(authSvc apimiddleware.TokenValidator) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.AuthOptional(authSvc))
		r.Get("/", h.list)
		r.Get("/{id}", h.getByID)
	})

	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.AuthRequired(authSvc))
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})

	return r
}

// @Summary List items catalog
// @Tags items
// @Produce json
// @Param page       query int    false "Page number (default: 1)"
// @Param per_page   query int    false "Items per page (default: 20, max: 100)"
// @Param category_id query string false "Filter by category UUID"
// @Param category_ids query string false "Filter by one or more category UUIDs, comma-separated"
// @Param city_uuid  query string false "Filter by city UUID"
// @Param geoname_id query int    false "Filter by GeoNames ID (alternative to city_uuid)"
// @Param condition  query string false "Filter by condition" Enums(new, used)
// @Param price_min  query int    false "Min price in minor units"
// @Param price_max  query int    false "Max price in minor units"
// @Param seller_id  query string false "Filter by seller UUID"
// @Param search     query string false "Search by title"
// @Param statuses   query string false "Comma-separated statuses (e.g. published,archived). Non-published only allowed for own seller profile."
// @Success 200 {object} ItemListResponse
// @Router /api/v1/items [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	pp, err := pagination.ParseParams(r)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, err.Error()))
		return
	}

	q, err := api.DecodeQueryAndValidate[ItemListQuery](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}
	if q.PriceMin != nil && q.PriceMax != nil && *q.PriceMin > *q.PriceMax {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "price_min must be less than or equal to price_max"))
		return
	}

	cityID, err := cityresolver.ResolveCityID(r.Context(), q.CityID, q.GeonameID, h.cityRepo)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, err.Error()))
		return
	}

	itemSortAllowed := itemsort.Allowed{
		"created_at": "i.created_at",
		"title":      "i.title",
		"price":      "i.price_amount",
		"city_uuid":  "i.city_id",
		"status":     "i.status",
	}
	order, err := itemsort.Parse(r, itemSortAllowed, "created_at")
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, err.Error()))
		return
	}

	statuses, requireOwnership, authErr := resolveStatuses(r, q)
	if authErr != nil {
		api.Error(w, h.logger, authErr)
		return
	}

	filter := itemquery.ListFilter{
		CategoryIDs:      categoryIDsFromQuery(q),
		CityID:           cityID,
		Condition:        q.Condition,
		PriceMin:         q.PriceMin,
		PriceMax:         q.PriceMax,
		SellerID:         q.SellerID,
		Search:           q.Search,
		Page:             pp.Page,
		PerPage:          pp.PerPage,
		OrderBy:          order.ToSQL(),
		Statuses:         statuses,
		RequireOwnership: requireOwnership,
	}

	result, svcErr := h.readSvc.List(r.Context(), filter, currentViewerID(r))
	if svcErr != nil {
		api.Error(w, h.logger, svcErr)
		return
	}

	items := make([]ItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, MapItemToResponse(item))
	}

	api.JSON(w, http.StatusOK, ItemListResponse{
		Data: items,
		Pagination: api.PaginationResponse{
			Page:       result.Pagination.Page,
			PerPage:    result.Pagination.PerPage,
			Total:      result.Pagination.Total,
			TotalPages: result.Pagination.TotalPages,
		},
	})
}

// @Summary Get item by ID or slug
// @Tags items
// @Produce json
// @Param id path string true "Item UUID or slug"
// @Success 200 {object} ItemSingleResponse
// @Failure 404 {object} api.ProblemDetails
// @Router /api/v1/items/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "id")

	roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)
	isAdmin := hasRole(roles, model.RoleAdmin)

	var item *itemquery.Item
	var svcErr error

	if id, err := uuid.Parse(param); err == nil {
		item, svcErr = h.readSvc.GetByID(r.Context(), id, currentViewerID(r), isAdmin)
	} else if reSlug.MatchString(param) {
		item, svcErr = h.readSvc.GetBySlug(r.Context(), param, currentViewerID(r), isAdmin)
	} else {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "invalid item id or slug"))
		return
	}

	if svcErr != nil {
		api.Error(w, h.logger, svcErr)
		return
	}

	api.JSON(w, http.StatusOK, ItemSingleResponse{
		Data: MapItemToResponse(*item),
	})
}

// @Summary Create item
// @Description Any authenticated user can create an item. Status is immediately 'published' (no moderation in MVP). Optional thumbnail_id sets the cover image from any uploaded item photo (independent of photo_ids).
// @Tags items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateItemRequest true "Item data"
// @Success 201 {object} ItemSingleResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/items [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "user not authenticated"))
		return
	}

	req, err := api.DecodeAndValidate[CreateItemRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if req.CityID == nil && req.GeonameID == nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "either city_uuid or geoname_id must be provided"))
		return
	}

	city, err := cityresolver.ResolveCity(r.Context(), req.CityID, req.GeonameID, h.cityRepo)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, err.Error()))
		return
	}

	if city == nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "city_uuid or geoname_id could not be resolved"))
		return
	}

	roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)
	isAdmin := hasRole(roles, model.RoleAdmin)

	item, svcErr := h.writeSvc.Create(r.Context(), itemservice.CreateInput{
		Title:       req.Title,
		CategoryID:  req.CategoryID,
		Description: req.Description,
		Condition:   req.Condition,
		Tags:        req.Tags,
		PhotoIDs:    req.PhotoIDs,
		ThumbnailID: req.ThumbnailID,
		CityID:      city.ID,
		CitySlug:    city.Slug,
		CreatedBy:   userID,
		IsAdmin:     isAdmin,
		Price:       mapPriceInput(req.Price),
	})
	if svcErr != nil {
		api.Error(w, h.logger, svcErr)
		return
	}

	view, err := h.readSvc.GetByID(r.Context(), item.ID, &userID, isAdmin)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusCreated, ItemSingleResponse{
		Data: MapItemToResponse(*view),
	})
}

// @Summary Update item
// @Description Creator or admin can update. Admins are identified by ROLE_ADMIN in their JWT. The thumbnail_id is independent of photo_ids and can reference any uploaded item photo. To clear thumbnail, send thumbnail_id=null.
// @Tags items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id   path string             true "Item UUID"
// @Param body body UpdateItemRequest true "Fields to update"
// @Success 200 {object} ItemSingleResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Failure 403 {object} api.ProblemDetails
// @Failure 404 {object} api.ProblemDetails
// @Router /api/v1/items/{id} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "invalid item id"))
		return
	}

	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "user not authenticated"))
		return
	}

	roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)
	isAdmin := hasRole(roles, model.RoleAdmin)

	req, err := api.DecodeAndValidate[UpdateItemRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	cityID, err := cityresolver.ResolveCityID(r.Context(), req.CityID, req.GeonameID, h.cityRepo)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, err.Error()))
		return
	}

	item, svcErr := h.writeSvc.Update(r.Context(), id, itemservice.UpdateInput{
		Title:          req.Title,
		CategoryID:     req.CategoryID,
		Description:    req.Description,
		Condition:      req.Condition,
		Tags:           req.Tags,
		PhotoIDs:       req.PhotoIDs,
		ThumbnailID:    req.ThumbnailID,
		ThumbnailIDSet: req.ThumbnailIDSet,
		CityID:         cityID,
		Status:         req.Status,
		Price:          mapPriceInput(req.Price),
		RequestedBy:    userID,
		IsAdmin:        isAdmin,
	})
	if svcErr != nil {
		api.Error(w, h.logger, svcErr)
		return
	}

	view, err := h.readSvc.GetByID(r.Context(), item.ID, &userID, isAdmin)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, ItemSingleResponse{
		Data: MapItemToResponse(*view),
	})
}

// @Summary Delete item
// @Description Creator or admin can delete.
// @Tags items
// @Produce json
// @Security BearerAuth
// @Param id path string true "Item UUID"
// @Success 200 {object} map[string]any
// @Failure 401 {object} api.ProblemDetails
// @Failure 403 {object} api.ProblemDetails
// @Failure 404 {object} api.ProblemDetails
// @Router /api/v1/items/{id} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "invalid item id"))
		return
	}

	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "user not authenticated"))
		return
	}

	roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)
	isAdmin := hasRole(roles, model.RoleAdmin)

	if svcErr := h.writeSvc.Delete(r.Context(), itemservice.DeleteInput{
		ID:          id,
		RequestedBy: userID,
		IsAdmin:     isAdmin,
	}); svcErr != nil {
		api.Error(w, h.logger, svcErr)
		return
	}

	api.JSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"deleted": true}})
}

func hasRole(roles []string, target model.UserRole) bool {
	return slices.Contains(roles, string(target))
}

func mapPriceInput(p *PriceRequest) *itemservice.PriceInput {
	if p == nil {
		return nil
	}
	return &itemservice.PriceInput{Amount: p.Amount, Currency: p.Currency}
}

func currentViewerID(r *http.Request) *uuid.UUID {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		return nil
	}

	return &userID
}

func categoryIDsFromQuery(q ItemListQuery) []uuid.UUID {
	categoryIDs := make([]uuid.UUID, 0, len(q.CategoryIDs)+1)
	if q.CategoryID != nil {
		categoryIDs = append(categoryIDs, *q.CategoryID)
	}
	categoryIDs = append(categoryIDs, q.CategoryIDs...)
	return categoryIDs
}

// resolveStatuses enforces status access rules:
//   - no statuses requested → nil, nil (repo defaults to ["published"])
//   - only "published" → always allowed, no ownership check needed
//   - any non-published status → requires auth + seller_id; admin bypasses ownership check;
//     non-admin gets a RequireOwnership UUID so the service verifies seller_id == viewer
func resolveStatuses(r *http.Request, q ItemListQuery) (statuses []string, requireOwnership *uuid.UUID, appErr *apierr.AppError) {
	if q.Status != "" {
		statuses = append(statuses, q.Status)
	}
	statuses = append(statuses, q.Statuses...)
	if len(statuses) == 0 {
		return nil, nil, nil
	}

	onlyPublished := true
	for _, status := range statuses {
		if status != string(model.ItemStatusPublished) {
			onlyPublished = false
			break
		}
	}
	if onlyPublished {
		return statuses, nil, nil
	}

	viewerID, authed := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !authed {
		return nil, nil, apierr.New(apierr.ErrForbidden, "authentication required to filter by non-published statuses")
	}

	roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)
	if hasRole(roles, model.RoleAdmin) {
		return statuses, nil, nil
	}

	if q.SellerID == nil {
		return nil, nil, apierr.New(apierr.ErrForbidden, "seller_id is required when filtering by non-published statuses")
	}

	return statuses, &viewerID, nil
}

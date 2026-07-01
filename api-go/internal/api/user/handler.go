package user

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	authservice "github.com/dionisvl/avi/api-go/internal/service/auth"
	userservice "github.com/dionisvl/avi/api-go/internal/service/user"
)

// ProfileRouter is implemented by any handler that exposes profile sub-routes.
type ProfileRouter interface {
	Routes(authSvc apimiddleware.TokenValidator) chi.Router
}

type Handler struct {
	authSvc        authservice.Service
	userSvc        userservice.Service
	logger         *slog.Logger
	profileRouters map[string]ProfileRouter
}

func NewHandler(authSvc authservice.Service, userSvc userservice.Service, logger *slog.Logger) *Handler {
	return &Handler{
		authSvc:        authSvc,
		userSvc:        userSvc,
		logger:         logger,
		profileRouters: map[string]ProfileRouter{},
	}
}

// WithProfileRouter mounts a sub-handler under /profile/<name>.
func (h *Handler) WithProfileRouter(name string, r ProfileRouter) *Handler {
	h.profileRouters[name] = r
	return h
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(apimiddleware.AuthRequired(h.authSvc))
	r.Get("/me", h.getMe)
	r.Patch("/me", h.updateMe)
	r.Delete("/me", h.deleteMe)
	for name, pr := range h.profileRouters {
		r.Mount("/profile/"+name, pr.Routes(h.authSvc))
	}
	return r
}

type MeResponse struct {
	ID            string                `json:"id"`
	Email         string                `json:"email"`
	Roles         []string              `json:"roles"`
	Name          string                `json:"name"`
	AvatarURL     string                `json:"avatar_url"`
	HasProfile    bool                  `json:"has_profile"`
	EmailVerified bool                  `json:"email_verified"`
	Preferences   model.UserPreferences `json:"preferences"`
	CreatedAt     time.Time             `json:"created_at"`
}

// Beta decision: {} → 400 (nothing to update), {"name": null} or {"name": ""} → 200 (clears name).
// When more fields are added, absent fields will simply be ignored.
type UpdateMeRequest struct {
	Name        *string             `json:"name"        validate:"omitempty,max=100"`
	Preferences *PreferencesRequest `json:"preferences"`
}

type PreferencesRequest struct {
	CategoryID *string `json:"category_id" validate:"omitempty,uuid"`
	CityID     *string `json:"city_id"     validate:"omitempty,uuid"`
	Condition  *string `json:"condition"   validate:"omitempty,oneof=new used"`
	PriceMin   *int64  `json:"price_min"   validate:"omitempty,min=0"`
	PriceMax   *int64  `json:"price_max"   validate:"omitempty,min=0"`
	Search     *string `json:"search"      validate:"omitempty,max=100"`
}

type DeleteMeRequest struct {
	Password string `json:"password" validate:"required"`
}

// @Summary Get current user info
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MeResponse
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/user/me [get]
func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	user, err := h.userSvc.GetMe(r.Context(), userID)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)

	api.JSON(w, http.StatusOK, MeResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		Roles:         roles,
		Name:          user.Name,
		AvatarURL:     user.AvatarURL,
		HasProfile:    user.Name != "",
		EmailVerified: user.IsEmailVerified,
		Preferences:   user.Preferences,
		CreatedAt:     user.CreatedAt,
	})
}

// @Summary Update current user profile
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateMeRequest true "Profile data"
// @Success 200 {object} MeResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/user/me [patch]
func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "Failed to read request body"))
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "Invalid JSON in request body"))
		return
	}
	if len(raw) == 0 {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "at least one field must be provided"))
		return
	}

	var req UpdateMeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "Invalid JSON in request body"))
		return
	}
	if _, hasName := raw["name"]; hasName && req.Name != nil && len(*req.Name) > 100 {
		api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "name must be at most 100 characters"))
		return
	}

	// {"name": null} → clear name (empty string)
	var namePtr *string
	if _, hasName := raw["name"]; hasName {
		if req.Name != nil {
			namePtr = req.Name
		} else {
			empty := ""
			namePtr = &empty
		}
	}

	var prefsPtr *model.UserPreferences
	if req.Preferences != nil {
		p := req.Preferences
		if p.CategoryID == nil && p.CityID == nil && p.Condition == nil && p.PriceMin == nil && p.PriceMax == nil && p.Search == nil {
			api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "preferences must contain at least one field"))
			return
		}
		if p.PriceMin != nil && p.PriceMax != nil && *p.PriceMin > *p.PriceMax {
			api.Error(w, h.logger, apierr.New(apierr.ErrValidation, "price_min must be less than or equal to price_max"))
			return
		}
		if err := api.ValidateStruct(p); err != nil {
			api.Error(w, h.logger, err)
			return
		}
		prefsPtr = &model.UserPreferences{
			CategoryID: p.CategoryID,
			CityID:     p.CityID,
			Condition:  p.Condition,
			PriceMin:   p.PriceMin,
			PriceMax:   p.PriceMax,
			Search:     p.Search,
		}
	}

	_, err = h.userSvc.UpdateMe(r.Context(), userID, userservice.UpdateMeInput{
		Name:        namePtr,
		Preferences: prefsPtr,
	})
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	// Override only the cached user entry so GetMe reloads from DB while preserving request-scoped values.
	ctx := context.WithValue(r.Context(), apimiddleware.ContextKeyUser, (*model.User)(nil))
	var user *model.User
	user, err = h.userSvc.GetMe(ctx, userID)
	if err != nil {
		api.Error(w, h.logger, api.AsAppError(err))
		return
	}

	roles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)

	api.JSON(w, http.StatusOK, MeResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		Roles:         roles,
		Name:          user.Name,
		AvatarURL:     user.AvatarURL,
		HasProfile:    user.Name != "",
		EmailVerified: user.IsEmailVerified,
		Preferences:   user.Preferences,
		CreatedAt:     user.CreatedAt,
	})
}

// @Summary Delete current user account
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body DeleteMeRequest true "Current password confirmation"
// @Success 204
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/user/me [delete]
func (h *Handler) deleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	req, err := api.DecodeAndValidate[DeleteMeRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.userSvc.DeleteMe(r.Context(), userID, req.Password); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

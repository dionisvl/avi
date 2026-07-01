package auth

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	"github.com/dionisvl/avi/api-go/internal/config"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	authservice "github.com/dionisvl/avi/api-go/internal/service/auth"
)

func hasRole(roles []string, role model.UserRole) bool {
	return slices.Contains(roles, string(role))
}

type Handler struct {
	svc            authservice.Service
	rateLimitRPS   float64
	rateLimitBurst int
	trustedProxies []string
	logger         *slog.Logger
}

func NewHandler(svc authservice.Service, cfg config.AuthConfig, appCfg config.AppConfig, logger *slog.Logger) *Handler {
	return &Handler{
		svc:            svc,
		rateLimitRPS:   cfg.RateLimitRPS,
		rateLimitBurst: cfg.RateLimitBurst,
		trustedProxies: appCfg.TrustedProxies,
		logger:         logger,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(apimiddleware.RateLimit(h.rateLimitRPS, h.rateLimitBurst, h.trustedProxies...))

	r.With(apimiddleware.AuthOptional(h.svc)).Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/refresh", h.refresh)
	r.Post("/verify-email", h.verifyEmail)
	r.Post("/resend-verification", h.resendVerification)

	r.Route("/reset-password", func(r chi.Router) {
		r.Post("/request", h.resetPasswordRequest)
		r.Post("/confirm", h.resetPasswordConfirm)
		r.Post("/set", h.resetPasswordSet)
	})

	// Authenticated endpoints
	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.AuthRequired(h.svc))
		r.Post("/logout", h.logout)
		r.Post("/change-password", h.changePassword)
	})

	return r
}

// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "Registration data"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 409 {object} api.ProblemDetails
// @Router /api/v1/auth/register [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[RegisterRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if req.EmailVerified {
		if _, authed := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID); !authed {
			api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "Valid admin token required to use email_verified flag"))
			return
		}
		callerRoles, _ := r.Context().Value(apimiddleware.ContextKeyRoles).([]string)
		if !hasRole(callerRoles, model.RoleAdmin) {
			api.Error(w, h.logger, apierr.New(apierr.ErrForbidden, "Only admins can use email_verified flag"))
			return
		}
	}

	emailVerified := req.EmailVerified

	out, err := h.svc.Register(r.Context(), authservice.RegisterInput{
		Email:         req.Email,
		Password:      req.Password,
		EmailVerified: emailVerified,
		Locale:        requestLocale(req.Locale, r.Header.Get("Accept-Language")),
	})
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusCreated, RegisterResponse{
		ID:    out.ID.String(),
		Email: out.Email,
		Roles: out.Roles,
	})
}

func requestLocale(bodyLocale, acceptLanguage string) string {
	if bodyLocale != "" {
		return bodyLocale
	}

	tag, _, _ := strings.Cut(acceptLanguage, ",")
	tag = strings.TrimSpace(strings.ToLower(tag))
	tag, _, _ = strings.Cut(tag, ";")
	lang, _, _ := strings.Cut(tag, "-")
	return lang
}

// @Summary Login
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[LoginRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	tokens, err := h.svc.Login(r.Context(), authservice.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// @Summary Refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "Refresh token"
// @Success 200 {object} RefreshResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/auth/refresh [post]
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[RefreshRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	tokens, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// @Summary Request password reset
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResetPasswordRequestReq true "Email"
// @Success 200 {object} map[string]string
// @Failure 400 {object} api.ProblemDetails
// @Router /api/v1/auth/reset-password/request [post]
func (h *Handler) resetPasswordRequest(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[ResetPasswordRequestReq](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.svc.RequestPasswordReset(r.Context(), req.Email); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Check your email"})
}

// @Summary Confirm password reset code
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResetPasswordConfirmReq true "Email and code"
// @Success 200 {object} map[string]string
// @Failure 400 {object} api.ProblemDetails
// @Router /api/v1/auth/reset-password/confirm [post]
func (h *Handler) resetPasswordConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[ResetPasswordConfirmReq](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.svc.ConfirmPasswordReset(r.Context(), req.Email, req.Code); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Code confirmed"})
}

// @Summary Set new password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResetPasswordSetReq true "Email, code and new password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} api.ProblemDetails
// @Router /api/v1/auth/reset-password/set [post]
func (h *Handler) resetPasswordSet(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[ResetPasswordSetReq](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.svc.SetNewPassword(r.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Password updated"})
}

// @Summary Change password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ChangePasswordRequest true "Current and new password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} api.ProblemDetails
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/auth/change-password [post]
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	req, err := api.DecodeAndValidate[ChangePasswordRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.svc.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// @Summary Logout current user from all devices
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 401 {object} api.ProblemDetails
// @Router /api/v1/auth/logout [post]
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "User not authenticated"))
		return
	}

	if err := h.svc.Logout(r.Context(), userID); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Logged out from all devices"})
}

// @Summary Resend email verification code
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResendVerificationRequest true "Email and optional locale"
// @Success 200 {object} map[string]string
// @Failure 429 {object} api.ProblemDetails
// @Router /api/v1/auth/resend-verification [post]
func (h *Handler) resendVerification(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[ResendVerificationRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.svc.ResendVerification(r.Context(), req.Email, requestLocale(req.Locale, r.Header.Get("Accept-Language"))); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Verification code sent"})
}

// @Summary Verify email
// @Tags auth
// @Accept json
// @Produce json
// @Param body body VerifyEmailRequest true "Email and verification code"
// @Success 200 {object} map[string]string
// @Failure 400 {object} api.ProblemDetails
// @Router /api/v1/auth/verify-email [post]
func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[VerifyEmailRequest](r)
	if err != nil {
		api.Error(w, h.logger, err)
		return
	}

	if err := h.svc.VerifyEmail(r.Context(), req.Email, req.Code); err != nil {
		api.Error(w, h.logger, err)
		return
	}

	api.JSON(w, http.StatusOK, map[string]string{"message": "Email verified successfully"})
}

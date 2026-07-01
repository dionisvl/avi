package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/service/auth"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userID"
	ContextKeyEmail  contextKey = "email"
	ContextKeyRoles  contextKey = "roles"
	ContextKeyUser   contextKey = "user"
)

type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, tokenString string) (*model.User, *auth.Claims, error)
}

// ProblemDetails is RFC 9457 response
type ProblemDetails struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

// AuthOptional extracts userID from Bearer token if present, but does not reject anonymous requests.
// Downstream handlers can check ctx.Value(ContextKeyUserID) — nil means anonymous.
func AuthOptional(tokenSvc TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					if user, claims, err := tokenSvc.ValidateAccessToken(r.Context(), parts[1]); err == nil {
						ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
						ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
						ctx = context.WithValue(ctx, ContextKeyRoles, claims.Roles)
						ctx = context.WithValue(ctx, ContextKeyUser, user)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AuthRequired(tokenSvc TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, apierr.New(apierr.ErrInvalidToken, "Missing authorization header"))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeAuthError(w, apierr.New(apierr.ErrInvalidToken, "Invalid authorization format"))
				return
			}

			user, claims, err := tokenSvc.ValidateAccessToken(r.Context(), parts[1])
			if err != nil {
				writeAuthError(w, apierr.New(apierr.ErrInvalidToken, "Invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextKeyRoles, claims.Roles)
			ctx = context.WithValue(ctx, ContextKeyUser, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRoles(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, ok := r.Context().Value(ContextKeyRoles).([]string)
			if !ok {
				writeAuthError(w, apierr.New(apierr.ErrForbidden, "Insufficient permissions"))
				return
			}

			allowed := false
			roleMap := make(map[string]struct{}, len(roles))
			for _, role := range roles {
				roleMap[role] = struct{}{}
			}

			for _, requiredRole := range requiredRoles {
				if _, exists := roleMap[requiredRole]; exists {
					allowed = true
					break
				}
			}

			if !allowed {
				writeAuthError(w, apierr.New(apierr.ErrForbidden, "Insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(w http.ResponseWriter, appErr *apierr.AppError) {
	resp := ProblemDetails{
		Status: appErr.Status,
		Title:  appErr.Title,
		Detail: appErr.Detail,
		Code:   string(appErr.Code),
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(appErr.Status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

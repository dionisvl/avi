package middleware

import (
	"net/http"

	"github.com/dionisvl/avi/api-go/internal/platform/locale"
)

// Locale resolves the request locale from ?locale= query param or Accept-Language header
// (respecting q-weights) and stores it in context via platform/locale.
func Locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc := locale.Resolve(r.URL.Query().Get("locale"), r.Header.Get("Accept-Language"))
		next.ServeHTTP(w, r.WithContext(locale.WithLocale(r.Context(), loc)))
	})
}

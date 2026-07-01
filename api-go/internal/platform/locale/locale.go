package locale

import (
	"context"
	"strconv"
	"strings"
)

type contextKey struct{}

const Default = "en"

var Supported = map[string]struct{}{
	"en": {},
	"ru": {},
}

func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, contextKey{}, locale)
}

func FromCtx(ctx context.Context) string {
	if l, ok := ctx.Value(contextKey{}).(string); ok && l != "" {
		return l
	}
	return Default
}

// Resolve picks the best supported locale.
// Priority: ?locale= query param (if supported) > Accept-Language header (by q-weight) > Default.
func Resolve(queryLocale, acceptLanguage string) string {
	if _, ok := Supported[queryLocale]; ok {
		return queryLocale
	}

	best := ""
	bestQ := -1.0

	for part := range strings.SplitSeq(acceptLanguage, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tag := part
		q := 1.0
		if before, after, ok := strings.Cut(part, ";"); ok {
			tag = strings.TrimSpace(before)
			for param := range strings.SplitSeq(after, ";") {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "q=") {
					if v, err := strconv.ParseFloat(param[2:], 64); err == nil {
						q = v
					}
				}
			}
		}

		if q <= 0 {
			continue
		}

		// Normalise: en-US → en, ru-RU → ru
		lang := strings.ToLower(tag)
		if i := strings.IndexByte(lang, '-'); i >= 0 {
			lang = lang[:i]
		}

		if _, ok := Supported[lang]; ok && q > bestQ {
			best = lang
			bestQ = q
		}
	}

	if best != "" {
		return best
	}
	return Default
}

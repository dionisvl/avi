package pagination

import (
	"errors"
	"net/http"
	"strconv"
)

var (
	ErrInvalidPage    = errors.New("page must be a positive integer")
	ErrInvalidPerPage = errors.New("per_page must be a positive integer")
)

// ParseParams extracts and validates page/per_page from query string.
func ParseParams(r *http.Request) (Params, error) {
	q := r.URL.Query()

	page := DefaultPage
	perPage := DefaultPerPage

	if raw := q.Get("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			return Params{}, ErrInvalidPage
		}
		page = v
	}

	if raw := q.Get("per_page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			return Params{}, ErrInvalidPerPage
		}
		perPage = v
	}

	return NewParams(page, perPage), nil
}

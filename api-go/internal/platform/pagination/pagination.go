package pagination

import "math"

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Params holds validated pagination parameters.
type Params struct {
	Page    int
	PerPage int
}

// Meta is the pagination metadata included in list responses.
type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewParams returns validated Params with defaults and caps applied.
func NewParams(page, perPage int) Params {
	if page < 1 {
		page = DefaultPage
	}
	switch {
	case perPage < 1:
		perPage = DefaultPerPage
	case perPage > MaxPerPage:
		perPage = MaxPerPage
	}
	return Params{Page: page, PerPage: perPage}
}

// Limit returns the SQL LIMIT value.
func (p Params) Limit() int {
	return p.PerPage
}

// Offset returns the SQL OFFSET value.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Meta calculates pagination metadata given the total number of records.
func (p Params) Meta(total int) Meta {
	if total < 0 {
		total = 0
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(p.PerPage)))
	}
	return Meta{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

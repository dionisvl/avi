package sort

import (
	"fmt"
	"net/http"
	"strings"
)

// Order represents a single ORDER BY clause.
type Order struct {
	Column string
	Desc   bool
}

// ToSQL returns the ORDER BY fragment, e.g. "created_at DESC".
func (o Order) ToSQL() string {
	if o.Desc {
		return o.Column + " DESC"
	}
	return o.Column + " ASC"
}

// Allowed maps public sort keys to their SQL column expressions.
// Each entity defines its own Allowed map and passes it to Parse.
type Allowed map[string]string

// Parse extracts and validates the ?sort=[-]key query parameter.
// The optional leading "-" means descending (e.g. sort=-created_at).
// If the param is absent or empty, defaultKey is used.
// Returns ErrInvalidSort if the key is not in allowed.
func Parse(r *http.Request, allowed Allowed, defaultKey string) (Order, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("sort"))
	if raw == "" {
		raw = defaultKey
	}

	desc := false
	key := raw
	if strings.HasPrefix(raw, "-") {
		desc = true
		key = raw[1:]
	}

	col, ok := allowed[key]
	if !ok {
		keys := make([]string, 0, len(allowed))
		for k := range allowed {
			keys = append(keys, k)
		}
		return Order{}, fmt.Errorf("sort must be one of: %s", strings.Join(keys, ", "))
	}

	return Order{Column: col, Desc: desc}, nil
}

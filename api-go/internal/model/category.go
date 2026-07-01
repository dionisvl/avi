package model

import (
	"slices"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID
	Slug      string
	ParentID  *uuid.UUID
	Names     map[string]string // locale -> name
	SortOrder int16
	IsActive  bool
	Name      string // resolved for requested locale (set by query layer), optional
}

// LocalizedName returns the name for the requested locale, falling back to "en" then any available name.
func (c *Category) LocalizedName(locale string) string {
	if c == nil || len(c.Names) == 0 {
		return ""
	}

	// Try exact locale match
	if name, ok := c.Names[locale]; ok {
		return name
	}

	// Fall back to English
	if name, ok := c.Names["en"]; ok {
		return name
	}

	// Return any available name (prefer iteration stability by sorting)
	keys := make([]string, 0, len(c.Names))
	for key := range c.Names {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if len(keys) > 0 {
		return c.Names[keys[0]]
	}

	return ""
}

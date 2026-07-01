package model

import (
	"github.com/google/uuid"
)

type City struct {
	ID         uuid.UUID
	Slug       string
	GeonameID  *int
	Names      map[string]string
	IsActive   bool
	Population int
}

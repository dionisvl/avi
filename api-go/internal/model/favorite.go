package model

import (
	"time"

	"github.com/google/uuid"
)

type Favorite struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ItemID    uuid.UUID
	CreatedAt time.Time
}

type FavoriteWithItem struct {
	Favorite
	Item ItemWithDetails
}

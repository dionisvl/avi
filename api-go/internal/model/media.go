package model

import (
	"time"

	"github.com/google/uuid"
)

type ItemPhoto struct {
	ID               uuid.UUID
	ItemID           *uuid.UUID // nullable: photo may be uploaded before the item is created
	UploaderID       *uuid.UUID // nullable for legacy rows; set on all new uploads
	Bucket           string
	ObjectKey        string
	MimeType         string
	SizeBytes        int64
	OriginalFilename string
	SortOrder        int16
	CreatedAt        time.Time
}

type UserAvatar struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Bucket           string
	ObjectKey        string
	MimeType         string
	SizeBytes        int64
	OriginalFilename string
	CreatedAt        time.Time
}

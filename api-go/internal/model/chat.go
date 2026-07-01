package model

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID            uuid.UUID
	UserA         uuid.UUID
	UserB         uuid.UUID
	CreatedAt     time.Time
	LastMessageAt time.Time
}

type ChatMessage struct {
	ID                  uuid.UUID
	ConversationID      uuid.UUID
	SenderID            uuid.UUID
	Body                *string
	AttachmentObjectKey *string
	AttachmentMIME      *string
	AttachmentSize      *int64
	CreatedAt           time.Time
}

type ChatRead struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	LastReadAt     time.Time
}

package chat

import (
	"context"
	"log/slog"
	"maps"

	"github.com/google/uuid"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	chatrepo "github.com/dionisvl/avi/api-go/internal/repository/chat"
)

type SendMessageInput struct {
	ConversationID      uuid.UUID
	SenderID            uuid.UUID
	Body                *string
	AttachmentObjectKey *string
	AttachmentMIME      *string
	AttachmentSize      *int64
}

type Service interface {
	OpenConversation(ctx context.Context, userID, peerID uuid.UUID) (*model.Conversation, error)
	SendMessage(ctx context.Context, in SendMessageInput) (*model.ChatMessage, error)
	MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error
	IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
	EnsureParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
}

type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type Hub interface {
	SendToUser(userID uuid.UUID, msg any)
}

type service struct {
	repo       chatrepo.Repository
	userReader UserReader
	hub        Hub
	s3BaseURL  string
	logger     *slog.Logger
}

func New(repo chatrepo.Repository, userReader UserReader, hub Hub, s3BaseURL string, logger *slog.Logger) Service {
	return &service{
		repo:       repo,
		userReader: userReader,
		hub:        hub,
		s3BaseURL:  s3BaseURL,
		logger:     logger,
	}
}

func (s *service) OpenConversation(ctx context.Context, userID, peerID uuid.UUID) (*model.Conversation, error) {
	if userID == peerID {
		return nil, apierr.New(apierr.ErrBadRequest, "cannot create chat with yourself")
	}

	peer, err := s.userReader.GetByID(ctx, peerID)
	if err != nil {
		if chatrepo.IsNotFound(err) {
			return nil, apierr.New(apierr.ErrNotFound, "user not found")
		}
		return nil, err
	}
	if peer == nil {
		return nil, apierr.New(apierr.ErrNotFound, "user not found")
	}

	conv, err := s.repo.UpsertConversation(ctx, userID, peerID)
	if err != nil {
		return nil, err
	}

	return conv, nil
}

func (s *service) SendMessage(ctx context.Context, in SendMessageInput) (*model.ChatMessage, error) {
	isParticipant, err := s.repo.EnsureParticipant(ctx, in.ConversationID, in.SenderID)
	if err != nil {
		if chatrepo.IsNotFound(err) {
			return nil, apierr.New(apierr.ErrNotFound, "conversation not found")
		}
		return nil, err
	}
	if !isParticipant {
		return nil, apierr.New(apierr.ErrForbidden, "not a participant of this conversation")
	}

	if in.Body == nil && in.AttachmentObjectKey == nil {
		return nil, apierr.New(apierr.ErrBadRequest, "message must contain body or attachment")
	}

	msg := &model.ChatMessage{
		ConversationID:      in.ConversationID,
		SenderID:            in.SenderID,
		Body:                in.Body,
		AttachmentObjectKey: in.AttachmentObjectKey,
		AttachmentMIME:      in.AttachmentMIME,
		AttachmentSize:      in.AttachmentSize,
	}

	if err := s.repo.InsertMessage(ctx, msg); err != nil {
		return nil, err
	}

	conv, err := s.repo.GetConversation(ctx, in.ConversationID)
	if err != nil && !chatrepo.IsNotFound(err) {
		s.logger.Warn("failed to load conversation for ws notify, message saved but notification skipped",
			slog.String("conversation_id", in.ConversationID.String()),
			slog.String("error", err.Error()),
		)
	}

	if conv != nil {
		var recipientID uuid.UUID
		if conv.UserA == in.SenderID {
			recipientID = conv.UserB
		} else {
			recipientID = conv.UserA
		}

		var attachmentURL *string
		if msg.AttachmentObjectKey != nil {
			url := s.s3BaseURL + "/" + *msg.AttachmentObjectKey
			attachmentURL = &url
		}

		baseMsg := map[string]any{
			"conversation_id": msg.ConversationID,
			"id":              msg.ID,
			"sender_id":       msg.SenderID,
			"body":            msg.Body,
			"attachment_url":  attachmentURL,
			"attachment_mime": msg.AttachmentMIME,
			"attachment_size": msg.AttachmentSize,
			"created_at":      msg.CreatedAt,
		}

		recipientMsg := make(map[string]any, len(baseMsg)+1)
		maps.Copy(recipientMsg, baseMsg)
		recipientMsg["is_mine"] = false
		s.hub.SendToUser(recipientID, recipientMsg)

		senderMsg := make(map[string]any, len(baseMsg)+2)
		maps.Copy(senderMsg, baseMsg)
		senderMsg["is_mine"] = true
		senderMsg["status"] = "sent"
		s.hub.SendToUser(in.SenderID, senderMsg)
	}

	return msg, nil
}

func (s *service) MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		if chatrepo.IsNotFound(err) {
			return apierr.New(apierr.ErrNotFound, "conversation not found")
		}
		return apierr.Wrap(err, "failed to get conversation")
	}

	var peerID uuid.UUID
	if conv.UserA == userID {
		peerID = conv.UserB
	} else if conv.UserB == userID {
		peerID = conv.UserA
	} else {
		return apierr.New(apierr.ErrForbidden, "not a participant of this conversation")
	}

	readAt, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return apierr.Wrap(err, "failed to mark read")
	}

	s.hub.SendToUser(peerID, map[string]any{
		"type":            "read",
		"conversation_id": conversationID,
		"read_at":         readAt,
	})

	return nil
}

func (s *service) IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	return s.repo.IsParticipant(ctx, conversationID, userID)
}

func (s *service) EnsureParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	isParticipant, err := s.repo.EnsureParticipant(ctx, conversationID, userID)
	if err != nil {
		if chatrepo.IsNotFound(err) {
			return false, apierr.New(apierr.ErrNotFound, "conversation not found")
		}
		return false, err
	}
	return isParticipant, nil
}

package chatview

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	chatrepo "github.com/dionisvl/avi/api-go/internal/repository/chat"
)

type ConversationView struct {
	ID                  uuid.UUID
	PeerID              uuid.UUID
	PeerName            string
	PeerType            string
	PeerAvatarURL       *string
	LastMessagePreview  *string
	LastMessageHasPhoto bool
	LastMessageAt       time.Time
	UnreadCount         int
}

type MessageView struct {
	ID             uuid.UUID
	SenderID       uuid.UUID
	Body           *string
	IsMine         bool
	Status         *string // "sent" | "read"; nil for incoming messages
	AttachmentURL  *string
	AttachmentMIME *string
	AttachmentSize *int64
	CreatedAt      time.Time
}

type Service interface {
	ListConversations(ctx context.Context, userID uuid.UUID) ([]ConversationView, error)
	GetConversation(ctx context.Context, conversationID, userID uuid.UUID) (*ConversationView, error)
	ListMessages(ctx context.Context, conversationID, userID uuid.UUID, before time.Time, limit int) ([]MessageView, error)
	TotalUnread(ctx context.Context, userID uuid.UUID) (int, error)
}

type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type service struct {
	repo       chatrepo.Repository
	s3BaseURL  string
	userReader UserReader
}

func New(repo chatrepo.Repository, s3BaseURL string, userReader UserReader) Service {
	return &service{
		repo:       repo,
		s3BaseURL:  s3BaseURL,
		userReader: userReader,
	}
}

func (s *service) ListConversations(ctx context.Context, userID uuid.UUID) ([]ConversationView, error) {
	convs, err := s.repo.ListConversations(ctx, userID)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to list conversations")
	}

	if len(convs) == 0 {
		return []ConversationView{}, nil
	}

	convIDs := make([]uuid.UUID, len(convs))
	for i, c := range convs {
		convIDs[i] = c.ID
	}

	unreadCounts, err := s.repo.UnreadCounts(ctx, userID, convIDs)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to get unread counts")
	}

	lastMessages, err := s.repo.LastMessages(ctx, convIDs)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to get last messages")
	}

	views := make([]ConversationView, 0, len(convs))
	for _, conv := range convs {
		var peerID uuid.UUID
		if conv.UserA == userID {
			peerID = conv.UserB
		} else {
			peerID = conv.UserA
		}

		peer, err := s.userReader.GetByID(ctx, peerID)
		if err != nil {
			return nil, apierr.Wrap(err, "failed to load conversation peer")
		}

		peerName := s.peerName(peer)
		peerType := "user"

		var avatarURL *string
		if peer != nil && peer.AvatarURL != "" {
			avatarURL = &peer.AvatarURL
		}

		lm := lastMessages[conv.ID]
		views = append(views, ConversationView{
			ID:                  conv.ID,
			PeerID:              peerID,
			PeerName:            peerName,
			PeerType:            peerType,
			PeerAvatarURL:       avatarURL,
			LastMessagePreview:  lm.Body,
			LastMessageHasPhoto: lm.HasAttachment,
			LastMessageAt:       conv.LastMessageAt,
			UnreadCount:         unreadCounts[conv.ID],
		})
	}

	return views, nil
}

func (s *service) GetConversation(ctx context.Context, conversationID, userID uuid.UUID) (*ConversationView, error) {
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		if chatrepo.IsNotFound(err) {
			return nil, apierr.New(apierr.ErrNotFound, "conversation not found")
		}
		return nil, apierr.Wrap(err, "failed to get conversation")
	}

	var peerID uuid.UUID
	if conv.UserA == userID {
		peerID = conv.UserB
	} else {
		peerID = conv.UserA
	}

	peer, err := s.userReader.GetByID(ctx, peerID)
	if err != nil || peer == nil {
		return nil, apierr.New(apierr.ErrNotFound, "peer user not found")
	}

	peerName := s.peerName(peer)
	peerType := "user"

	var avatarURL *string
	if peer.AvatarURL != "" {
		avatarURL = &peer.AvatarURL
	}

	lastMessages, err := s.repo.LastMessages(ctx, []uuid.UUID{conversationID})
	if err != nil {
		return nil, apierr.Wrap(err, "failed to get last message")
	}

	unreadCounts, err := s.repo.UnreadCounts(ctx, userID, []uuid.UUID{conversationID})
	if err != nil {
		return nil, apierr.Wrap(err, "failed to get unread count")
	}

	lm := lastMessages[conversationID]
	view := &ConversationView{
		ID:                  conv.ID,
		PeerID:              peerID,
		PeerName:            peerName,
		PeerType:            peerType,
		PeerAvatarURL:       avatarURL,
		LastMessagePreview:  lm.Body,
		LastMessageHasPhoto: lm.HasAttachment,
		LastMessageAt:       conv.LastMessageAt,
		UnreadCount:         unreadCounts[conversationID],
	}

	return view, nil
}

func (s *service) peerName(peer *model.User) string {
	if peer == nil {
		return ""
	}

	if name := strings.TrimSpace(peer.Name); name != "" {
		return name
	}

	if email := strings.TrimSpace(peer.Email); email != "" {
		return email
	}

	return peer.ID.String()
}

const (
	statusSent = "sent"
	statusRead = "read"
)

func (s *service) ListMessages(ctx context.Context, conversationID, userID uuid.UUID, before time.Time, limit int) ([]MessageView, error) {
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		if chatrepo.IsNotFound(err) {
			return nil, apierr.New(apierr.ErrNotFound, "conversation not found")
		}
		return nil, apierr.Wrap(err, "failed to get conversation")
	}

	var peerID uuid.UUID
	if conv.UserA == userID {
		peerID = conv.UserB
	} else if conv.UserB == userID {
		peerID = conv.UserA
	} else {
		return nil, apierr.New(apierr.ErrForbidden, "not a participant of this conversation")
	}

	msgs, err := s.repo.ListMessages(ctx, conversationID, before, limit)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to list messages")
	}

	if len(msgs) == 0 {
		return []MessageView{}, nil
	}

	peerLastReadAt, err := s.repo.GetPeerLastReadAt(ctx, conversationID, peerID)
	if err != nil {
		return nil, apierr.Wrap(err, "failed to get peer read status")
	}

	slices.Reverse(msgs)

	views := make([]MessageView, len(msgs))
	for i, msg := range msgs {
		view := MessageView{
			ID:             msg.ID,
			SenderID:       msg.SenderID,
			Body:           msg.Body,
			IsMine:         msg.SenderID == userID,
			AttachmentMIME: msg.AttachmentMIME,
			AttachmentSize: msg.AttachmentSize,
			CreatedAt:      msg.CreatedAt,
		}

		if msg.SenderID == userID {
			msgStatus := statusSent
			if peerLastReadAt != nil && !msg.CreatedAt.After(*peerLastReadAt) {
				msgStatus = statusRead
			}
			view.Status = &msgStatus
		}

		if msg.AttachmentObjectKey != nil {
			url := s.s3BaseURL + "/" + *msg.AttachmentObjectKey
			view.AttachmentURL = &url
		}

		views[i] = view
	}

	return views, nil
}

func (s *service) TotalUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.repo.TotalUnreadConversations(ctx, userID)
	if err != nil {
		return 0, apierr.Wrap(err, "failed to get total unread")
	}
	return count, nil
}

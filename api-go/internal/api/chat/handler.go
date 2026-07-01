package chat

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/imageproc"
	chatquery "github.com/dionisvl/avi/api-go/internal/query/chatview"
	chatservice "github.com/dionisvl/avi/api-go/internal/service/chat"
	mediasvc "github.com/dionisvl/avi/api-go/internal/service/media"
)

type Handler struct {
	writeSvc  chatservice.Service
	readSvc   chatquery.Service
	mediaSvc  mediasvc.Service
	hub       *Hub
	s3BaseURL string
	wsOrigins []string
	logger    *slog.Logger
}

func NewHandler(writeSvc chatservice.Service, readSvc chatquery.Service, mediaSvc mediasvc.Service, hub *Hub, s3BaseURL string, wsOrigins []string, logger *slog.Logger) *Handler {
	return &Handler{
		writeSvc:  writeSvc,
		readSvc:   readSvc,
		mediaSvc:  mediaSvc,
		hub:       hub,
		s3BaseURL: s3BaseURL,
		wsOrigins: wsOrigins,
		logger:    logger,
	}
}

func (h *Handler) Routes(authSvc apimiddleware.TokenValidator) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.AuthRequired(authSvc))

		r.Get("/unread-total", h.unreadTotal)
		r.Post("/conversations", h.openConversation)
		r.Get("/conversations", h.listConversations)
		r.Get("/conversations/{id}/messages", h.listMessages)
		r.Post("/conversations/{id}/messages", h.sendMessage)
		r.Post("/conversations/{id}/read", h.markRead)
	})

	r.Get("/conversations/{id}/ws", h.serveWS(authSvc))

	return r
}

// Request/Response types

type OpenConversationRequest struct {
	PeerUserID uuid.UUID `json:"peer_user_id" validate:"required"`
}

type ConversationResponse struct {
	ID                  uuid.UUID `json:"id"`
	PeerID              uuid.UUID `json:"peer_id"`
	PeerName            string    `json:"peer_name"`
	PeerType            string    `json:"peer_type"`
	PeerAvatarURL       *string   `json:"peer_avatar_url,omitempty"`
	LastMessagePreview  *string   `json:"last_message_preview,omitempty"`
	LastMessageHasPhoto bool      `json:"last_message_has_photo"`
	LastMessageAt       time.Time `json:"last_message_at"`
	UnreadCount         int       `json:"unread_count"`
}

type MessageResponse struct {
	ID             uuid.UUID `json:"id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Body           *string   `json:"body,omitempty"`
	IsMine         bool      `json:"is_mine"`
	Status         *string   `json:"status,omitempty"` // "sent" | "read"; present only for outgoing messages
	AttachmentURL  *string   `json:"attachment_url,omitempty"`
	AttachmentMIME *string   `json:"attachment_mime,omitempty"`
	AttachmentSize *int64    `json:"attachment_size,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Handlers

// @Summary Open or get a conversation with a user
// @Tags chat
// @Accept json
// @Produce json
// @Param body body OpenConversationRequest true "Request"
// @Success 200 {object} ConversationResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /chat/conversations [post]
func (h *Handler) openConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "user not authenticated"))
		return
	}

	req, appErr := api.DecodeAndValidate[OpenConversationRequest](r)
	if appErr != nil {
		h.handleErr(w, appErr, "invalid request")
		return
	}

	conv, err := h.writeSvc.OpenConversation(r.Context(), userID, req.PeerUserID)
	if err != nil {
		h.handleErr(w, err, "failed to open conversation")
		return
	}

	convView, err := h.readSvc.GetConversation(r.Context(), conv.ID, userID)
	if err != nil {
		h.handleErr(w, err, "failed to load conversation details")
		return
	}

	resp := ConversationResponse{
		ID:                  convView.ID,
		PeerID:              convView.PeerID,
		PeerName:            convView.PeerName,
		PeerType:            convView.PeerType,
		PeerAvatarURL:       convView.PeerAvatarURL,
		LastMessagePreview:  convView.LastMessagePreview,
		LastMessageHasPhoto: convView.LastMessageHasPhoto,
		LastMessageAt:       convView.LastMessageAt,
		UnreadCount:         convView.UnreadCount,
	}

	api.JSON(w, http.StatusOK, resp)
}

// @Summary List conversations for authenticated user
// @Tags chat
// @Produce json
// @Success 200 {array} ConversationResponse
// @Failure 401 {object} map[string]interface{}
// @Router /chat/conversations [get]
func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "user not authenticated"))
		return
	}

	convs, err := h.readSvc.ListConversations(r.Context(), userID)
	if err != nil {
		h.handleErr(w, err, "failed to list conversations")
		return
	}

	responses := make([]ConversationResponse, len(convs))
	for i, conv := range convs {
		responses[i] = ConversationResponse{
			ID:                  conv.ID,
			PeerID:              conv.PeerID,
			PeerName:            conv.PeerName,
			PeerType:            conv.PeerType,
			PeerAvatarURL:       conv.PeerAvatarURL,
			LastMessagePreview:  conv.LastMessagePreview,
			LastMessageHasPhoto: conv.LastMessageHasPhoto,
			LastMessageAt:       conv.LastMessageAt,
			UnreadCount:         conv.UnreadCount,
		}
	}

	api.JSON(w, http.StatusOK, responses)
}

// @Summary List messages in a conversation
// @Tags chat
// @Produce json
// @Param id path string true "Conversation ID"
// @Param before query string false "Message created_at for keyset pagination"
// @Param limit query int false "Limit (default 50, max 100)"
// @Success 200 {array} MessageResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /chat/conversations/{id}/messages [get]
func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "user not authenticated"))
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid conversation id"))
		return
	}

	beforeTime := time.Now()
	if before := r.URL.Query().Get("before"); before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid before: must be RFC3339"))
			return
		}
		beforeTime = t
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	msgs, err := h.readSvc.ListMessages(r.Context(), convID, userID, beforeTime, limit)
	if err != nil {
		h.handleErr(w, err, "failed to list messages")
		return
	}

	responses := make([]MessageResponse, len(msgs))
	for i, msg := range msgs {
		responses[i] = MessageResponse{
			ID:             msg.ID,
			SenderID:       msg.SenderID,
			Body:           msg.Body,
			IsMine:         msg.IsMine,
			Status:         msg.Status,
			AttachmentURL:  msg.AttachmentURL,
			AttachmentMIME: msg.AttachmentMIME,
			AttachmentSize: msg.AttachmentSize,
			CreatedAt:      msg.CreatedAt,
		}
	}

	api.JSON(w, http.StatusOK, responses)
}

// @Summary Send a message (text and/or attachment)
// @Tags chat
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Conversation ID"
// @Param body formData string false "Message body"
// @Param file formData file false "Attachment file"
// @Success 201 {object} MessageResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /chat/conversations/{id}/messages [post]
func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	const maxBodySize int64 = 10 << 20

	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "user not authenticated"))
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid conversation id"))
		return
	}

	var body *string
	var attachmentObjectKey, attachmentMIME *string
	var attachmentSize *int64

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	if err := r.ParseMultipartForm(maxBodySize); err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrRequestTooLarge, "request too large"))
		return
	}

	if bodyValue := r.FormValue("body"); bodyValue != "" {
		body = &bodyValue
	}

	file, _, err := r.FormFile("file")
	if err == nil {
		defer func() {
			_ = file.Close()
		}()

		data := make([]byte, 512)
		n, _ := file.Read(data)
		mimeType := imageproc.DetectMIME(data[:n])

		if !imageproc.IsSupported(mimeType) {
			api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "unsupported file type"))
			return
		}

		_, _ = file.Seek(0, 0)

		uploadRes, err := h.mediaSvc.UploadChatAttachment(r.Context(), mediasvc.UploadChatAttachmentInput{
			ConversationID: convID,
			Body:           file,
		})
		if err != nil {
			api.Error(w, h.logger, apierr.Wrap(err, "failed to upload attachment"))
			return
		}

		webpMIME := "image/webp"
		attachmentObjectKey = &uploadRes.ObjectKey
		attachmentMIME = &webpMIME
		attachmentSize = &uploadRes.Size
	} else if err != http.ErrMissingFile {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid file upload"))
		return
	}

	if body == nil && attachmentObjectKey == nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "message must contain body or attachment"))
		return
	}

	msg, err := h.writeSvc.SendMessage(r.Context(), chatservice.SendMessageInput{
		ConversationID:      convID,
		SenderID:            userID,
		Body:                body,
		AttachmentObjectKey: attachmentObjectKey,
		AttachmentMIME:      attachmentMIME,
		AttachmentSize:      attachmentSize,
	})
	if err != nil {
		h.handleErr(w, err, "failed to send message")
		return
	}

	sent := "sent"
	resp := MessageResponse{
		ID:             msg.ID,
		SenderID:       msg.SenderID,
		Body:           msg.Body,
		IsMine:         msg.SenderID == userID,
		Status:         &sent,
		AttachmentURL:  nil,
		AttachmentMIME: msg.AttachmentMIME,
		AttachmentSize: msg.AttachmentSize,
		CreatedAt:      msg.CreatedAt,
	}

	if msg.AttachmentObjectKey != nil {
		url := h.s3BaseURL + "/" + *msg.AttachmentObjectKey
		resp.AttachmentURL = &url
	}

	api.JSON(w, http.StatusCreated, resp)
}

// @Summary Mark conversation as read
// @Tags chat
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 204
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /chat/conversations/{id}/read [post]
func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "user not authenticated"))
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid conversation id"))
		return
	}

	if err := h.writeSvc.MarkRead(r.Context(), convID, userID); err != nil {
		h.handleErr(w, err, "failed to mark as read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary WebSocket connection for real-time chat
// @Tags chat
// @Produce json
// @Param id path string true "Conversation ID"
// @Param token query string true "Access token"
// @Success 101
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /chat/conversations/{id}/ws [get]
func (h *Handler) serveWS(authSvc apimiddleware.TokenValidator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "missing token"))
			return
		}

		user, claims, err := authSvc.ValidateAccessToken(r.Context(), token)
		if err != nil || user == nil || claims == nil {
			api.Error(w, h.logger, apierr.New(apierr.ErrInvalidToken, "invalid token"))
			return
		}

		userID := user.ID

		convID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid conversation id"))
			return
		}

		isParticipant, err := h.writeSvc.EnsureParticipant(r.Context(), convID, userID)
		if err != nil {
			h.handleErr(w, err, "failed to check participation")
			return
		}
		if !isParticipant {
			api.Error(w, h.logger, apierr.New(apierr.ErrForbidden, "not a participant of this conversation"))
			return
		}

		ServeWS(h.hub, w, r, userID, convID, h.wsOrigins, h.logger)
	}
}

// @Summary Get total count of conversations with unread messages
// @Tags chat
// @Produce json
// @Success 200 {object} map[string]int
// @Failure 401 {object} map[string]interface{}
// @Router /chat/unread-total [get]
func (h *Handler) unreadTotal(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "user not authenticated"))
		return
	}

	total, err := h.readSvc.TotalUnread(r.Context(), userID)
	if err != nil {
		h.handleErr(w, err, "failed to get unread total")
		return
	}

	api.JSON(w, http.StatusOK, map[string]int{"total": total})
}

func (h *Handler) handleErr(w http.ResponseWriter, err error, fallback string) {
	if appErr, ok := err.(*apierr.AppError); ok {
		api.Error(w, h.logger, appErr)
	} else {
		api.Error(w, h.logger, apierr.New(apierr.ErrInternal, fallback))
	}
}

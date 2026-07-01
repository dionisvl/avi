package chat

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
)

type LastMessage struct {
	Body          *string
	HasAttachment bool
}

type Repository interface {
	UpsertConversation(ctx context.Context, userA, userB uuid.UUID) (*model.Conversation, error)
	GetConversation(ctx context.Context, id uuid.UUID) (*model.Conversation, error)
	IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
	EnsureParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
	ListConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error)

	InsertMessage(ctx context.Context, msg *model.ChatMessage) error
	ListMessages(ctx context.Context, conversationID uuid.UUID, before time.Time, limit int) ([]model.ChatMessage, error)
	LastMessages(ctx context.Context, convIDs []uuid.UUID) (map[uuid.UUID]LastMessage, error)

	MarkRead(ctx context.Context, conversationID, userID uuid.UUID) (time.Time, error)
	UnreadCounts(ctx context.Context, userID uuid.UUID, convIDs []uuid.UUID) (map[uuid.UUID]int, error)
	GetPeerLastReadAt(ctx context.Context, conversationID, peerID uuid.UUID) (*time.Time, error)
	TotalUnreadConversations(ctx context.Context, userID uuid.UUID) (int, error)
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) UpsertConversation(ctx context.Context, userA, userB uuid.UUID) (*model.Conversation, error) {
	if userA.String() > userB.String() {
		userA, userB = userB, userA
	}

	var conv model.Conversation
	err := r.db.QueryRow(ctx,
		`INSERT INTO conversations (user_a, user_b)
		 VALUES ($1, $2)
		 ON CONFLICT (user_a, user_b) DO NOTHING
		 RETURNING id, user_a, user_b, created_at, last_message_at`,
		userA, userB,
	).Scan(&conv.ID, &conv.UserA, &conv.UserB, &conv.CreatedAt, &conv.LastMessageAt)

	if err == pgx.ErrNoRows {
		err = r.db.QueryRow(ctx,
			`SELECT id, user_a, user_b, created_at, last_message_at FROM conversations WHERE user_a = $1 AND user_b = $2`,
			userA, userB,
		).Scan(&conv.ID, &conv.UserA, &conv.UserB, &conv.CreatedAt, &conv.LastMessageAt)
	}

	return &conv, err
}

func (r *repository) GetConversation(ctx context.Context, id uuid.UUID) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.QueryRow(ctx,
		`SELECT id, user_a, user_b, created_at, last_message_at FROM conversations WHERE id = $1`,
		id,
	).Scan(&conv.ID, &conv.UserA, &conv.UserB, &conv.CreatedAt, &conv.LastMessageAt)
	return &conv, err
}

func (r *repository) IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	var isParticipant bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM conversations WHERE id = $1 AND ($2::uuid IN (user_a, user_b)))`,
		conversationID, userID,
	).Scan(&isParticipant)
	return isParticipant, err
}

func (r *repository) EnsureParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	var isParticipant bool
	err := r.db.QueryRow(ctx,
		`SELECT ($2::uuid IN (user_a, user_b)) FROM conversations WHERE id = $1`,
		conversationID, userID,
	).Scan(&isParticipant)
	return isParticipant, err
}

func (r *repository) ListConversations(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_a, user_b, created_at, last_message_at
		 FROM conversations
		 WHERE user_a = $1 OR user_b = $1
		 ORDER BY last_message_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	convs := make([]model.Conversation, 0)
	for rows.Next() {
		var conv model.Conversation
		if err := rows.Scan(&conv.ID, &conv.UserA, &conv.UserB, &conv.CreatedAt, &conv.LastMessageAt); err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}

	return convs, rows.Err()
}

func (r *repository) InsertMessage(ctx context.Context, msg *model.ChatMessage) error {
	tb, ok := r.db.(dbtx.TxBeginner)
	if !ok {
		return errors.New("db does not support transactions")
	}

	tx, err := tb.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	err = tx.QueryRow(ctx,
		`INSERT INTO chat_messages (conversation_id, sender_id, body, attachment_object_key, attachment_mime, attachment_size)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		msg.ConversationID, msg.SenderID, msg.Body, msg.AttachmentObjectKey, msg.AttachmentMIME, msg.AttachmentSize,
	).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE conversations SET last_message_at = NOW() WHERE id = $1`,
		msg.ConversationID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *repository) ListMessages(ctx context.Context, conversationID uuid.UUID, before time.Time, limit int) ([]model.ChatMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, conversation_id, sender_id, body, attachment_object_key, attachment_mime, attachment_size, created_at
		 FROM chat_messages
		 WHERE conversation_id = $1 AND created_at < $2
		 ORDER BY created_at DESC
		 LIMIT $3`,
		conversationID, before, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := make([]model.ChatMessage, 0)
	for rows.Next() {
		var msg model.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Body, &msg.AttachmentObjectKey, &msg.AttachmentMIME, &msg.AttachmentSize, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	return msgs, rows.Err()
}

func (r *repository) MarkRead(ctx context.Context, conversationID, userID uuid.UUID) (time.Time, error) {
	var readAt time.Time
	err := r.db.QueryRow(ctx,
		`INSERT INTO chat_reads (conversation_id, user_id, last_read_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (conversation_id, user_id) DO UPDATE SET last_read_at = NOW()
		 RETURNING last_read_at`,
		conversationID, userID,
	).Scan(&readAt)
	return readAt, err
}

func (r *repository) UnreadCounts(ctx context.Context, userID uuid.UUID, convIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			cm.conversation_id,
			COUNT(*) as unread_count
		 FROM chat_messages cm
		 LEFT JOIN chat_reads cr ON cr.conversation_id = cm.conversation_id AND cr.user_id = $1
		 WHERE cm.conversation_id = ANY($2)
			AND cm.sender_id != $1
			AND cm.created_at > COALESCE(cr.last_read_at, '1970-01-01')
		 GROUP BY cm.conversation_id`,
		userID, convIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var convID uuid.UUID
		var count int
		if err := rows.Scan(&convID, &count); err != nil {
			return nil, err
		}
		counts[convID] = count
	}

	for _, id := range convIDs {
		if _, exists := counts[id]; !exists {
			counts[id] = 0
		}
	}

	return counts, rows.Err()
}

func (r *repository) LastMessages(ctx context.Context, convIDs []uuid.UUID) (map[uuid.UUID]LastMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (conversation_id) conversation_id, body, attachment_object_key IS NOT NULL
		 FROM chat_messages
		 WHERE conversation_id = ANY($1)
		 ORDER BY conversation_id, created_at DESC`,
		convIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]LastMessage, len(convIDs))
	for rows.Next() {
		var convID uuid.UUID
		var lm LastMessage
		if err := rows.Scan(&convID, &lm.Body, &lm.HasAttachment); err != nil {
			return nil, err
		}
		result[convID] = lm
	}
	return result, rows.Err()
}

func (r *repository) TotalUnreadConversations(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT cm.conversation_id)
		 FROM chat_messages cm
		 JOIN conversations c ON c.id = cm.conversation_id
		 LEFT JOIN chat_reads cr ON cr.conversation_id = cm.conversation_id AND cr.user_id = $1
		 WHERE (c.user_a = $1 OR c.user_b = $1)
		   AND cm.sender_id != $1
		   AND cm.created_at > COALESCE(cr.last_read_at, '1970-01-01')`,
		userID,
	).Scan(&count)
	return count, err
}

func (r *repository) GetPeerLastReadAt(ctx context.Context, conversationID, peerID uuid.UUID) (*time.Time, error) {
	var lastReadAt time.Time
	err := r.db.QueryRow(ctx,
		`SELECT last_read_at FROM chat_reads WHERE conversation_id = $1 AND user_id = $2`,
		conversationID, peerID,
	).Scan(&lastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lastReadAt, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

package chat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

const (
	sendBuffer = 64
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	conn *websocket.Conn
	send chan any
	once sync.Once
}

type Hub struct {
	logger *slog.Logger
	mu     sync.RWMutex
	users  map[uuid.UUID]map[*Client]struct{}
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		logger: logger,
		users:  make(map[uuid.UUID]map[*Client]struct{}),
	}
}

func (h *Hub) Register(userID uuid.UUID, conn *websocket.Conn) *Client {
	c := &Client{
		conn: conn,
		send: make(chan any, sendBuffer),
	}

	h.mu.Lock()
	if h.users[userID] == nil {
		h.users[userID] = make(map[*Client]struct{})
	}
	h.users[userID][c] = struct{}{}
	h.mu.Unlock()

	go h.writePump(c)
	return c
}

func (h *Hub) Unregister(userID uuid.UUID, c *Client) {
	h.mu.Lock()
	if clients, ok := h.users[userID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.users, userID)
		}
	}
	h.mu.Unlock()

	c.once.Do(func() { close(c.send) })
}

func (h *Hub) SendToUser(userID uuid.UUID, msg any) {
	h.mu.RLock()
	clients, ok := h.users[userID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	clientsCopy := make([]*Client, 0, len(clients))
	for c := range clients {
		clientsCopy = append(clientsCopy, c)
	}
	h.mu.RUnlock()

	for _, c := range clientsCopy {
		select {
		case c.send <- msg:
		default:
			h.logger.Warn("client send buffer full, dropping")
			h.Unregister(userID, c)
			_ = c.conn.CloseNow()
		}
	}
}

func (h *Hub) writePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.CloseNow()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := wsjson.Write(ctx, c.conn, msg)
			cancel()
			if err != nil {
				h.logger.Debug("write failed, closing client", slog.String("error", err.Error()))
				return
			}
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

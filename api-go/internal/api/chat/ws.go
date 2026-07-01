package chat

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, userID, conversationID uuid.UUID, originPatterns []string, logger *slog.Logger) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPatterns,
	})
	if err != nil {
		logger.Error("websocket accept failed", slog.String("error", err.Error()))
		return
	}
	defer func() {
		_ = conn.CloseNow()
	}()

	conn.SetReadLimit(4 << 10)

	client := hub.Register(userID, conn)
	defer hub.Unregister(userID, client)

	readLoop(client, conversationID, logger)
}

func readLoop(client *Client, conversationID uuid.UUID, logger *slog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Server is push-only; we read solely to detect client disconnect and honor the read deadline.
	for {
		var msg map[string]any
		err := wsjson.Read(ctx, client.conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			logger.Debug("read error", slog.String("error", err.Error()))
			return
		}
	}
}

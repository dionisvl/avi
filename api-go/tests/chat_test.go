package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatOpenConversation_RequiresAuth(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChatOpenConversation_Idempotent(t *testing.T) {
	app := newTestApp(t)

	userAToken := registerVerifyAndLogin(t, app, "user-a-"+uuid.New().String()+"@test.local", "password123")
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req1 := httptest.NewRequest("POST", "/api/v1/chat/conversations",
		bytes.NewReader(body))
	req1.Header.Set("Authorization", "Bearer "+userAToken)
	w1 := httptest.NewRecorder()
	app.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code)

	var conv1 map[string]any
	err = json.Unmarshal(w1.Body.Bytes(), &conv1)
	require.NoError(t, err)
	convID1 := conv1["id"].(string)

	req2 := httptest.NewRequest("POST", "/api/v1/chat/conversations",
		bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+userAToken)
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var conv2 map[string]any
	err = json.Unmarshal(w2.Body.Bytes(), &conv2)
	require.NoError(t, err)
	convID2 := conv2["id"].(string)

	require.Equal(t, convID1, convID2, "idempotent: same conversation ID on second call")
}

func TestChatOpenConversation_CannotChatWithSelf(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")

	var userAID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userAEmail).Scan(&userAID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userAID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatSendMessage_NonParticipantCannotWrite(t *testing.T) {
	app := newTestApp(t)

	userAToken := registerVerifyAndLogin(t, app, "user-a-"+uuid.New().String()+"@test.local", "password123")
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")
	userCToken := registerVerifyAndLogin(t, app, "user-c-"+uuid.New().String()+"@test.local", "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	sendBody := bytes.NewBufferString("--boundary\r\n" +
		"Content-Disposition: form-data; name=\"body\"\r\n\r\n" +
		"test message\r\n" +
		"--boundary--\r\n")

	sendReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/messages",
		sendBody)
	sendReq.Header.Set("Authorization", "Bearer "+userCToken)
	sendReq.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	sendW := httptest.NewRecorder()
	app.ServeHTTP(sendW, sendReq)

	require.Equal(t, http.StatusForbidden, sendW.Code)
}

func TestChatWebSocket_SmokeTest_MessageDelivery(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	userBToken := registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userAID, userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userAEmail).Scan(&userAID)
	require.NoError(t, err)
	err = app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	messageMarker := uuid.New().String()

	wsServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		app.ServeHTTP(rw, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + wsServer.URL[4:] + "/api/v1/chat/conversations/" + convID + "/ws"

	msgChan := make(chan map[string]any, 1)
	errChan := make(chan error, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		wsURL = wsURL + "?token=" + url.QueryEscape(userBToken)

		opts := &websocket.DialOptions{
			HTTPHeader: http.Header{"Origin": []string{"http://example.com"}},
		}
		conn, resp, wsErr := websocket.Dial(ctx, wsURL, opts)
		if wsErr != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			errChan <- wsErr
			return
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		conn.SetReadLimit(32 << 20)

		var msg map[string]any
		wsErr = wsjson.Read(ctx, conn, &msg)
		if wsErr != nil {
			errChan <- wsErr
			return
		}
		msgChan <- msg
	}()

	time.Sleep(500 * time.Millisecond)

	sendBody := bytes.NewBufferString("--boundary\r\n" +
		"Content-Disposition: form-data; name=\"body\"\r\n\r\n" +
		"test: " + messageMarker + "\r\n" +
		"--boundary--\r\n")

	sendReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/messages",
		sendBody)
	sendReq.Header.Set("Authorization", "Bearer "+userAToken)
	sendReq.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	sendW := httptest.NewRecorder()
	app.ServeHTTP(sendW, sendReq)
	require.Equal(t, http.StatusCreated, sendW.Code)

	select {
	case err := <-errChan:
		t.Fatalf("WS read error: %v", err)
	case msg := <-msgChan:
		body, ok := msg["body"].(string)
		require.True(t, ok, "message has body field")
		require.Contains(t, body, messageMarker, "received message contains unique marker")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for WebSocket message delivery")
	}
}

func TestChatSendMessage_TextMessage_AppearsInHistory(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	testMessage := "Hello " + uuid.New().String()
	sendBody := bytes.NewBufferString("--boundary\r\n" +
		"Content-Disposition: form-data; name=\"body\"\r\n\r\n" +
		testMessage + "\r\n" +
		"--boundary--\r\n")

	sendReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/messages", sendBody)
	sendReq.Header.Set("Authorization", "Bearer "+userAToken)
	sendReq.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	sendW := httptest.NewRecorder()
	app.ServeHTTP(sendW, sendReq)
	require.Equal(t, http.StatusCreated, sendW.Code)

	historyReq := httptest.NewRequest("GET", "/api/v1/chat/conversations/"+convID+"/messages", nil)
	historyReq.Header.Set("Authorization", "Bearer "+userAToken)
	historyW := httptest.NewRecorder()
	app.ServeHTTP(historyW, historyReq)
	require.Equal(t, http.StatusOK, historyW.Code)

	var messages []map[string]any
	err = json.Unmarshal(historyW.Body.Bytes(), &messages)
	require.NoError(t, err)
	require.Greater(t, len(messages), 0, "history contains at least one message")

	found := false
	for _, msg := range messages {
		if msgBody, ok := msg["body"].(string); ok && msgBody == testMessage {
			found = true
			require.NotNil(t, msg["id"], "message has id")
			require.NotNil(t, msg["sender_id"], "message has sender_id")
			require.NotNil(t, msg["created_at"], "message has created_at")
			break
		}
	}
	require.True(t, found, "sent message appears in history")
}

func TestChatSendMessage_PhotoMessage(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	// Generate a valid 4x4 PNG in memory
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: 255, G: 128, B: 0, A: 255})
	var photoBuf bytes.Buffer
	require.NoError(t, png.Encode(&photoBuf, img))
	photoData := photoBuf.Bytes()

	sendBody := bytes.NewBufferString("--boundary\r\n" +
		"Content-Disposition: form-data; name=\"file\"; filename=\"test.png\"\r\n" +
		"Content-Type: image/png\r\n\r\n")
	sendBody.Write(photoData)
	sendBody.WriteString("\r\n--boundary--\r\n")

	sendReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/messages", sendBody)
	sendReq.Header.Set("Authorization", "Bearer "+userAToken)
	sendReq.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	sendW := httptest.NewRecorder()
	app.ServeHTTP(sendW, sendReq)
	require.Equal(t, http.StatusCreated, sendW.Code)

	var msgResp map[string]any
	err = json.Unmarshal(sendW.Body.Bytes(), &msgResp)
	require.NoError(t, err)
	require.Equal(t, "image/webp", msgResp["attachment_mime"], "attachment converted to webp")
	attachSize, ok := msgResp["attachment_size"].(float64)
	require.True(t, ok, "attachment_size is a number")
	require.Greater(t, attachSize, float64(0), "converted webp size must be non-zero")
	attachURL, ok := msgResp["attachment_url"].(string)
	require.True(t, ok, "attachment_url is a string")
	require.True(t, strings.HasSuffix(attachURL, ".webp"), "attachment_url must point to a .webp file")
}

func TestChatListMessages_KeysetPagination(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	for i := range 5 {
		sendBody := bytes.NewBufferString("--boundary\r\n" +
			"Content-Disposition: form-data; name=\"body\"\r\n\r\n" +
			"message-" + fmt.Sprintf("%d", i) + "\r\n" +
			"--boundary--\r\n")

		sendReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/messages", sendBody)
		sendReq.Header.Set("Authorization", "Bearer "+userAToken)
		sendReq.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		sendW := httptest.NewRecorder()
		app.ServeHTTP(sendW, sendReq)
		require.Equal(t, http.StatusCreated, sendW.Code)
		time.Sleep(10 * time.Millisecond)
	}

	historyReq := httptest.NewRequest("GET", "/api/v1/chat/conversations/"+convID+"/messages?limit=2", nil)
	historyReq.Header.Set("Authorization", "Bearer "+userAToken)
	historyW := httptest.NewRecorder()
	app.ServeHTTP(historyW, historyReq)
	require.Equal(t, http.StatusOK, historyW.Code)

	var messages []map[string]any
	err = json.Unmarshal(historyW.Body.Bytes(), &messages)
	require.NoError(t, err)
	require.Greater(t, len(messages), 0, "first page has messages")
	require.LessOrEqual(t, len(messages), 2, "first page respects limit")

	if len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		beforeTimestamp := lastMessage["created_at"].(string)

		pageReq := httptest.NewRequest("GET", "/api/v1/chat/conversations/"+convID+"/messages?limit=2&before="+url.QueryEscape(beforeTimestamp), nil)
		pageReq.Header.Set("Authorization", "Bearer "+userAToken)
		pageW := httptest.NewRecorder()
		app.ServeHTTP(pageW, pageReq)
		require.Equal(t, http.StatusOK, pageW.Code)

		var pageMessages []map[string]any
		err = json.Unmarshal(pageW.Body.Bytes(), &pageMessages)
		require.NoError(t, err)

		if len(pageMessages) > 0 {
			firstPageSet := make(map[string]bool)
			for _, msg := range messages {
				firstPageSet[msg["id"].(string)] = true
			}

			for _, msg := range pageMessages {
				require.False(t, firstPageSet[msg["id"].(string)], "no message overlap between pages")
			}
		}
	}
}

func TestChatWebSocket_RequiresToken(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	wsServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		app.ServeHTTP(rw, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + wsServer.URL[4:] + "/api/v1/chat/conversations/" + convID + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := &websocket.DialOptions{}
	conn, resp, err := websocket.Dial(ctx, wsURL, opts)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err, "WS connection without token should fail")
	if conn != nil {
		_ = conn.CloseNow()
	}
}

func TestChatWebSocket_RequiresValidToken(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	wsServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		app.ServeHTTP(rw, r)
	}))
	defer wsServer.Close()

	wsURL := "ws" + wsServer.URL[4:] + "/api/v1/chat/conversations/" + convID + "/ws?token=invalid_token_xyz"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := &websocket.DialOptions{}
	conn, resp, err := websocket.Dial(ctx, wsURL, opts)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	require.Error(t, err, "WS connection with invalid token should fail")
	if conn != nil {
		_ = conn.CloseNow()
	}
}

func TestChatUnreadTracking_IncrementAndReset(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	userBToken := registerVerifyAndLogin(t, app, userBEmail, "password123")

	var userAID, userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userAEmail).Scan(&userAID)
	require.NoError(t, err)
	err = app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	for i := range 3 {
		sendBody := bytes.NewBufferString("--boundary\r\n" +
			"Content-Disposition: form-data; name=\"body\"\r\n\r\n" +
			"unread-msg-" + fmt.Sprintf("%d", i) + "\r\n" +
			"--boundary--\r\n")

		sendReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/messages", sendBody)
		sendReq.Header.Set("Authorization", "Bearer "+userAToken)
		sendReq.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		sendW := httptest.NewRecorder()
		app.ServeHTTP(sendW, sendReq)
		require.Equal(t, http.StatusCreated, sendW.Code)
	}

	listConvReq := httptest.NewRequest("GET", "/api/v1/chat/conversations", nil)
	listConvReq.Header.Set("Authorization", "Bearer "+userBToken)
	listConvW := httptest.NewRecorder()
	app.ServeHTTP(listConvW, listConvReq)
	require.Equal(t, http.StatusOK, listConvW.Code)

	var convList []map[string]any
	err = json.Unmarshal(listConvW.Body.Bytes(), &convList)
	require.NoError(t, err)
	require.Greater(t, len(convList), 0, "user B has conversations")

	userBConv := convList[0]
	unreadCount := userBConv["unread_count"]
	require.NotNil(t, unreadCount, "conversation has unread_count field")
	require.Equal(t, float64(3), unreadCount, "unread count is 3 for messages from user A")

	markReadReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/read", bytes.NewReader([]byte(`{}`)))
	markReadReq.Header.Set("Authorization", "Bearer "+userBToken)
	markReadW := httptest.NewRecorder()
	app.ServeHTTP(markReadW, markReadReq)
	require.Equal(t, http.StatusNoContent, markReadW.Code)

	listConvReq2 := httptest.NewRequest("GET", "/api/v1/chat/conversations", nil)
	listConvReq2.Header.Set("Authorization", "Bearer "+userBToken)
	listConvW2 := httptest.NewRecorder()
	app.ServeHTTP(listConvW2, listConvReq2)
	require.Equal(t, http.StatusOK, listConvW2.Code)

	var convList2 []map[string]any
	err = json.Unmarshal(listConvW2.Body.Bytes(), &convList2)
	require.NoError(t, err)
	userBConv2 := convList2[0]
	unreadCount2 := userBConv2["unread_count"]
	require.Equal(t, float64(0), unreadCount2, "unread count is 0 after marking read")
}

func TestChatListMessages_NonParticipantCannotRead(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userCEmail := "user-c-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")
	userCToken := registerVerifyAndLogin(t, app, userCEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	listMessagesReq := httptest.NewRequest("GET", "/api/v1/chat/conversations/"+convID+"/messages", nil)
	listMessagesReq.Header.Set("Authorization", "Bearer "+userCToken)
	listMessagesW := httptest.NewRecorder()
	app.ServeHTTP(listMessagesW, listMessagesReq)

	require.Equal(t, http.StatusForbidden, listMessagesW.Code)
}

func TestChatMarkRead_NonParticipantCannotRead(t *testing.T) {
	app := newTestApp(t)

	userAEmail := "user-a-" + uuid.New().String() + "@test.local"
	userBEmail := "user-b-" + uuid.New().String() + "@test.local"
	userCEmail := "user-c-" + uuid.New().String() + "@test.local"
	userAToken := registerVerifyAndLogin(t, app, userAEmail, "password123")
	_ = registerVerifyAndLogin(t, app, userBEmail, "password123")
	userCToken := registerVerifyAndLogin(t, app, userCEmail, "password123")

	var userBID uuid.UUID
	err := app.tx.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, userBEmail).Scan(&userBID)
	require.NoError(t, err)

	reqBody := map[string]any{"peer_user_id": userBID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var conv map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &conv)
	require.NoError(t, err)
	convID := conv["id"].(string)

	markReadReq := httptest.NewRequest("POST", "/api/v1/chat/conversations/"+convID+"/read", bytes.NewReader([]byte(`{}`)))
	markReadReq.Header.Set("Authorization", "Bearer "+userCToken)
	markReadW := httptest.NewRecorder()
	app.ServeHTTP(markReadW, markReadReq)

	require.Equal(t, http.StatusForbidden, markReadW.Code)
}

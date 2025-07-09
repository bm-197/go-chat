package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/bm-197/go-chat/internal/models"
	"github.com/bm-197/go-chat/internal/store"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allowing all in dev
		},
	}
)

type WebSocketHandler struct {
	store      *store.RedisStore
	clients    map[string]*websocket.Conn
	clientsMux sync.RWMutex
}

func NewWebSocketHandler(store *store.RedisStore) *WebSocketHandler {
	return &WebSocketHandler{
		store:   store,
		clients: make(map[string]*websocket.Conn),
	}
}

type Message struct {
	Type     models.MessageType `json:"type"`               // "private", "group, or "broadcast"
	GroupID  string             `json:"group_id,omitempty"` // Required for group messages
	To       string             `json:"to,omitempty"`       // Required for private messages
	Content  string             `json:"content"`
	From     string             `json:"from,omitempty"`
	FromUser string             `json:"from_user,omitempty"`
}

func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	userID := c.Get("user_id").(string)
	username := c.Get("username").(string)

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}
	defer ws.Close()

	h.clientsMux.Lock()
	h.clients[userID] = ws
	h.clientsMux.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.listenPubSub(ctx, userID, ws)

	// Reads messages coming from the websocket client
	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("error unmarshaling message: %v", err)
			continue
		}

		msg.From = userID
		msg.FromUser = username

		if !msg.Type.IsValid() {
			log.Printf("invalid message type: %s", msg.Type)
			continue
		}

		switch msg.Type {
		case models.MessageTypePrivate:
			if err := h.handlePrivateMessage(msg); err != nil {
				log.Printf("error handling private message: %v", err)
			}

		case models.MessageTypeGroup:
			if err := h.handleGroupMessage(msg); err != nil {
				log.Printf("error handling group message: %v", err)
			}

		case models.MessageTypeBroadcast:
			if err := h.handleBroadcast(msg); err != nil {
				log.Printf("error handling broadcast: %v", err)
			}

		default:
			log.Printf("unknown message type: %s", msg.Type)
		}
	}

	h.clientsMux.Lock()
	delete(h.clients, userID)
	h.clientsMux.Unlock()

	return nil
}

func (h *WebSocketHandler) listenPubSub(ctx context.Context, userID string, ws *websocket.Conn) {
	groups, err := h.store.GetUserGroups(ctx, userID)
	if err != nil {
		log.Printf("failed to fetch user groups for subscriptions: %v", err)
	}

	channels := []string{
		"broadcast",
		fmt.Sprintf("user:%s", userID),
	}

	for _, g := range groups {
		channels = append(channels, fmt.Sprintf("group:%s", g.ID))
	}

	pubsub := h.store.Subscribe(ctx, channels...)

	defer pubsub.Close()

	var writeMu sync.Mutex

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			writeMu.Lock()
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				writeMu.Unlock()
				log.Printf("failed to write websocket message: %v", err)
				return
			}
			writeMu.Unlock()
		}
	}
}

func (h *WebSocketHandler) handlePrivateMessage(msg Message) error {
	recipient, err := h.store.GetUserByUsername(context.Background(), msg.To)
	if err != nil {
		return fmt.Errorf("recipient not found: %w", err)
	}

	message := &models.Message{
		Type:      models.MessageTypePrivate,
		Content:   msg.Content,
		FromID:    msg.From,
		FromUser:  msg.FromUser,
		ToID:      recipient.ID,
		Timestamp: time.Now(),
	}
	if err := h.store.SaveMessage(context.Background(), message); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	if err := h.store.PublishMessage(context.Background(), message); err != nil {
		log.Printf("failed to publish private message: %v", err)
	}

	return nil
}

func (h *WebSocketHandler) handleGroupMessage(msg Message) error {
	group, err := h.store.GetGroup(context.Background(), msg.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}

	isMember := slices.Contains(group.Members, msg.From)

	if !isMember {
		return fmt.Errorf("user is not a member of the group")
	}

	message := &models.Message{
		Type:      models.MessageTypeGroup,
		Content:   msg.Content,
		FromID:    msg.From,
		FromUser:  msg.FromUser,
		GroupID:   msg.GroupID,
		Timestamp: time.Now(),
	}
	if err := h.store.SaveMessage(context.Background(), message); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	if err := h.store.PublishMessage(context.Background(), message); err != nil {
		log.Printf("failed to publish group message: %v", err)
	}

	return nil
}

func (h *WebSocketHandler) handleBroadcast(msg Message) error {
	message := &models.Message{
		Type:      models.MessageTypeBroadcast,
		Content:   msg.Content,
		FromID:    msg.From,
		FromUser:  msg.FromUser,
		Timestamp: time.Now(),
	}
	if err := h.store.SaveMessage(context.Background(), message); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	if err := h.store.PublishMessage(context.Background(), message); err != nil {
		log.Printf("failed to publish broadcast message: %v", err)
	}

	return nil
}

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	defer func() {
		h.clientsMux.Lock()
		delete(h.clients, userID)
		h.clientsMux.Unlock()
	}()

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

	return nil
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

	h.clientsMux.RLock()
	recipientWS, ok := h.clients[recipient.ID]
	h.clientsMux.RUnlock()

	if ok {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		if err := recipientWS.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func (h *WebSocketHandler) handleGroupMessage(msg Message) error {
	group, err := h.store.GetGroup(context.Background(), msg.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}

	isMember := false
	for _, memberID := range group.Members {
		if memberID == msg.From {
			isMember = true
			break
		}
	}

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

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	h.clientsMux.RLock()
	for _, memberID := range group.Members {
		if ws, ok := h.clients[memberID]; ok {
			if err := ws.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				log.Printf("failed to send message to member %s: %v", memberID, err)
			}
		}
	}
	h.clientsMux.RUnlock()

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

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	h.clientsMux.RLock()
	for _, ws := range h.clients {
		if err := ws.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Printf("failed to send broadcast message: %v", err)
		}
	}
	h.clientsMux.RUnlock()

	return nil
}

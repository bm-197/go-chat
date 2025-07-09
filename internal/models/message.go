package models

import (
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypePrivate   MessageType = "private"
	MessageTypeGroup     MessageType = "group"
	MessageTypeBroadcast MessageType = "broadcast"
)

func (mt MessageType) String() string {
	return string(mt)
}

func (mt MessageType) IsValid() bool {
	switch mt {
	case MessageTypePrivate, MessageTypeGroup, MessageTypeBroadcast:
		return true
	default:
		return false
	}
}

type Message struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"` // "private", "group", or "broadcast"
	Content   string      `json:"content"`
	FromID    string      `json:"from_id"`
	FromUser  string      `json:"from_user"`
	ToID      string      `json:"to_id,omitempty"`    // For private
	GroupID   string      `json:"group_id,omitempty"` // For group
	Timestamp time.Time   `json:"timestamp"`
}

func NewMessage(msgType string, content, fromID, fromUser string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      MessageType(msgType),
		Content:   content,
		FromID:    fromID,
		FromUser:  fromUser,
		Timestamp: time.Now(),
	}
}

func (m *Message) SetPrivateRecipient(toID string) {
	m.ToID = toID
	m.GroupID = ""
}

func (m *Message) SetGroupRecipient(groupID string) {
	m.GroupID = groupID
	m.ToID = ""
}

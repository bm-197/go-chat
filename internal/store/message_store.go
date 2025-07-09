package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bm-197/go-chat/internal/models"
)

const (
	privateMessageKeyPrefix = "private_msg:"
	groupMessageKeyPrefix   = "group_msg:"
	broadcastKeyPrefix      = "broadcast"
)

func (s *RedisStore) SaveMessage(ctx context.Context, msg *models.Message) error {
	msgData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var key string
	switch msg.Type {
	case models.MessageTypePrivate:
		key1 := fmt.Sprintf("%s%s:%s", privateMessageKeyPrefix, msg.FromID, msg.ToID)
		key2 := fmt.Sprintf("%s%s:%s", privateMessageKeyPrefix, msg.ToID, msg.FromID)

		pipe := s.client.Pipeline()
		pipe.RPush(ctx, key1, msgData)
		pipe.RPush(ctx, key2, msgData)
		_, err = pipe.Exec(ctx)

	case models.MessageTypeGroup:
		key = fmt.Sprintf("%s%s", groupMessageKeyPrefix, msg.GroupID)
		err = s.client.RPush(ctx, key, msgData).Err()

	case models.MessageTypeBroadcast:
		key = broadcastKeyPrefix
		err = s.client.RPush(ctx, key, msgData).Err()

	default:
		return fmt.Errorf("invalid message type: %s", msg.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

func (s *RedisStore) GetPrivateMessages(ctx context.Context, user1, user2 string, limit int64) ([]*models.Message, error) {
	key := fmt.Sprintf("%s%s:%s", privateMessageKeyPrefix, user1, user2)
	return s.getMessages(ctx, key, limit)
}

func (s *RedisStore) GetGroupMessages(ctx context.Context, groupID string, limit int64) ([]*models.Message, error) {
	key := fmt.Sprintf("%s%s", groupMessageKeyPrefix, groupID)
	return s.getMessages(ctx, key, limit)
}

func (s *RedisStore) GetBroadcastMessages(ctx context.Context, limit int64) ([]*models.Message, error) {
	return s.getMessages(ctx, broadcastKeyPrefix, limit)
}

func (s *RedisStore) getMessages(ctx context.Context, key string, limit int64) ([]*models.Message, error) {
	msgDataList, err := s.client.LRange(ctx, key, -limit, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*models.Message, 0, len(msgDataList))
	for _, msgData := range msgDataList {
		var msg models.Message
		if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w", err)
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

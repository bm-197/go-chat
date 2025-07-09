package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/bm-197/go-chat/internal/models"
)

func (s *RedisStore) PublishMessage(ctx context.Context, msg *models.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var channel string
	switch msg.Type {
	case models.MessageTypePrivate:
		channel = fmt.Sprintf("user:%s", msg.ToID)
	case models.MessageTypeGroup:
		channel = fmt.Sprintf("group:%s", msg.GroupID)
	case models.MessageTypeBroadcast:
		channel = "broadcast"
	default:
		return fmt.Errorf("invalid message type: %s", msg.Type)
	}

	if err := s.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

func (s *RedisStore) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return s.client.Subscribe(ctx, channels...)
}

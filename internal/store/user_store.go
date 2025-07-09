package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bm-197/go-chat/internal/models"

	"github.com/go-redis/redis/v8"
)

const (
	userKeyPrefix     = "user:"
	usernameKeyPrefix = "username:"
)

func (s *RedisStore) SaveUser(ctx context.Context, user *models.User) error {
	userKey := fmt.Sprintf("%s%s", userKeyPrefix, user.ID)
	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	usernameKey := fmt.Sprintf("%s%s", usernameKeyPrefix, user.Username)

	exists, err := s.client.Exists(ctx, usernameKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists == 1 {
		return fmt.Errorf("username already exists")
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, userKey, userData, 0)
	pipe.Set(ctx, usernameKey, user.ID, 0)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

func (s *RedisStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	userKey := fmt.Sprintf("%s%s", userKeyPrefix, id)
	userData, err := s.client.Get(ctx, userKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user models.User
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	if user.Password == "" {
		return nil, fmt.Errorf("invalid user data: missing password hash")
	}

	return &user, nil
}

func (s *RedisStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	usernameKey := fmt.Sprintf("%s%s", usernameKeyPrefix, username)
	userID, err := s.client.Get(ctx, usernameKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	return s.GetUserByID(ctx, userID)
}

func (s *RedisStore) DeleteUser(ctx context.Context, user *models.User) error {
	userKey := fmt.Sprintf("%s%s", userKeyPrefix, user.ID)
	usernameKey := fmt.Sprintf("%s%s", usernameKeyPrefix, user.Username)

	pipe := s.client.Pipeline()
	pipe.Del(ctx, userKey)
	pipe.Del(ctx, usernameKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

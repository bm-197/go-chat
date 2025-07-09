package store

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(host, port string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
	}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

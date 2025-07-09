package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"github.com/bm-197/go-chat/internal/models"
)

const (
	groupKeyPrefix      = "group:"
	groupListKey        = "groups"
	userGroupsKeyPrefix = "user_groups:"
)

func (s *RedisStore) SaveGroup(ctx context.Context, group *models.Group) error {
	groupData, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group: %w", err)
	}

	key := fmt.Sprintf("%s%s", groupKeyPrefix, group.ID)
	pipe := s.client.Pipeline()
	pipe.Set(ctx, key, groupData, 0)
	pipe.SAdd(ctx, groupListKey, group.ID)

	for _, memberID := range group.Members {
		userGroupsKey := fmt.Sprintf("%s%s", userGroupsKeyPrefix, memberID)
		pipe.SAdd(ctx, userGroupsKey, group.ID)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save group: %w", err)
	}

	return nil
}

func (s *RedisStore) GetGroup(ctx context.Context, id string) (*models.Group, error) {
	key := fmt.Sprintf("%s%s", groupKeyPrefix, id)
	groupData, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	var group models.Group
	if err := json.Unmarshal(groupData, &group); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group: %w", err)
	}

	return &group, nil
}

func (s *RedisStore) GetAllGroups(ctx context.Context) ([]*models.Group, error) {
	groupIDs, err := s.client.SMembers(ctx, groupListKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get group IDs: %w", err)
	}
	groups := make([]*models.Group, 0, len(groupIDs))
	for _, id := range groupIDs {
		group, err := s.GetGroup(ctx, id)
		if err != nil {
			log.Printf("failed to get group %s: %v", id, err)
			continue
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (s *RedisStore) DeleteGroup(ctx context.Context, group *models.Group) error {
	key := fmt.Sprintf("%s%s", groupKeyPrefix, group.ID)
	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, groupListKey, group.ID)

	for _, memberID := range group.Members {
		userGroupsKey := fmt.Sprintf("%s%s", userGroupsKeyPrefix, memberID)
		pipe.SRem(ctx, userGroupsKey, group.ID)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

func (s *RedisStore) GetUserGroups(ctx context.Context, userID string) ([]*models.Group, error) {
	userGroupsKey := fmt.Sprintf("%s%s", userGroupsKeyPrefix, userID)
	groupIDs, err := s.client.SMembers(ctx, userGroupsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	groups := make([]*models.Group, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, err := s.GetGroup(ctx, groupID)
		if err != nil {
			continue
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (s *RedisStore) UpdateGroupMembers(ctx context.Context, group *models.Group, oldMembers []string) error {
	pipe := s.client.Pipeline()

	for _, memberID := range oldMembers {
		userGroupsKey := fmt.Sprintf("%s%s", userGroupsKeyPrefix, memberID)
		pipe.SRem(ctx, userGroupsKey, group.ID)
	}

	for _, memberID := range group.Members {
		userGroupsKey := fmt.Sprintf("%s%s", userGroupsKeyPrefix, memberID)
		pipe.SAdd(ctx, userGroupsKey, group.ID)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update group members: %w", err)
	}

	return nil
}

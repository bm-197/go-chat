package models

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	Members     []string  `json:"members"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewGroup(name, description, createdBy string) *Group {
	return &Group{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
		Members:     []string{createdBy}, // Creator is automatically a member
		CreatedAt:   time.Now(),
	}
}

func (g *Group) AddMember(userID string) bool {
	if slices.Contains(g.Members, userID) {
		return false
	}
	g.Members = append(g.Members, userID)
	return true
}

func (g *Group) RemoveMember(userID string) bool {
	index := slices.Index(g.Members, userID)
	if index == -1 {
		return false
	}
	g.Members = slices.Delete(g.Members, index, index+1)
	return true
}

func (g *Group) IsMember(userID string) bool {
	return slices.Contains(g.Members, userID)
}

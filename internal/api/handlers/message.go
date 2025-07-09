package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bm-197/go-chat/internal/models"
	"github.com/bm-197/go-chat/internal/store"
)

type MessageHandler struct {
	store *store.RedisStore
}

func NewMessageHandler(store *store.RedisStore) *MessageHandler {
	return &MessageHandler{
		store: store,
	}
}

type SendMessageRequest struct {
	Type    string `json:"type" validate:"required,oneof=private group broadcast"`
	Content string `json:"content" validate:"required"`
	ToUser  string `json:"to_user,omitempty"`
	ToGroup string `json:"to_group,omitempty"`
}

func (h *MessageHandler) SendMessage(c echo.Context) error {
	var req SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	userID := c.Get("user_id").(string)
	username := c.Get("username").(string)
	msg := models.NewMessage(req.Type, req.Content, userID, username)

	switch models.MessageType(req.Type) {
	case models.MessageTypePrivate:
		if req.ToUser == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "to_user is required for private messages")
		}
		_, err := h.store.GetUserByID(c.Request().Context(), req.ToUser)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "recipient not found")
		}
		msg.SetPrivateRecipient(req.ToUser)

	case models.MessageTypeGroup:
		if req.ToGroup == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "to_group is required for group messages")
		}
		group, err := h.store.GetGroup(c.Request().Context(), req.ToGroup)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "group not found")
		}
		if !group.IsMember(userID) {
			return echo.NewHTTPError(http.StatusForbidden, "not a member of this group")
		}
		msg.SetGroupRecipient(req.ToGroup)

	case models.MessageTypeBroadcast:
		// No additional validation needed for broadcast
	}

	if err := h.store.SaveMessage(c.Request().Context(), msg); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save message")
	}

	return c.JSON(http.StatusCreated, msg)
}

func (h *MessageHandler) GetPrivateMessages(c echo.Context) error {
	userID := c.Get("user_id").(string)
	otherUserID := c.Param("userID")
	limit := int64(50)

	messages, err := h.store.GetPrivateMessages(c.Request().Context(), userID, otherUserID, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get messages")
	}

	return c.JSON(http.StatusOK, messages)
}

func (h *MessageHandler) GetGroupMessages(c echo.Context) error {
	userID := c.Get("user_id").(string)
	groupID := c.Param("groupID")
	limit := int64(50)

	group, err := h.store.GetGroup(c.Request().Context(), groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}
	if !group.IsMember(userID) {
		return echo.NewHTTPError(http.StatusForbidden, "not a member of this group")
	}

	messages, err := h.store.GetGroupMessages(c.Request().Context(), groupID, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get messages")
	}

	return c.JSON(http.StatusOK, messages)
}

func (h *MessageHandler) GetBroadcastMessages(c echo.Context) error {
	limit := int64(50)

	messages, err := h.store.GetBroadcastMessages(c.Request().Context(), limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get messages")
	}

	return c.JSON(http.StatusOK, messages)
}

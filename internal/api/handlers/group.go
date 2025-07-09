package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bm-197/go-chat/internal/models"
	"github.com/bm-197/go-chat/internal/store"
)

type GroupHandler struct {
	store *store.RedisStore
}

func NewGroupHandler(store *store.RedisStore) *GroupHandler {
	return &GroupHandler{
		store: store,
	}
}

type CreateGroupRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

func (h *GroupHandler) CreateGroup(c echo.Context) error {
	var req CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	userID := c.Get("user_id").(string)
	group := models.NewGroup(req.Name, req.Description, userID)

	if err := h.store.SaveGroup(c.Request().Context(), group); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create group")
	}

	return c.JSON(http.StatusCreated, group)
}

func (h *GroupHandler) GetGroup(c echo.Context) error {
	groupID := c.Param("id")
	group, err := h.store.GetGroup(c.Request().Context(), groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	return c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) ListGroups(c echo.Context) error {
	groups, err := h.store.GetAllGroups(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get groups")
	}

	return c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) JoinGroup(c echo.Context) error {
	groupID := c.Param("id")
	userID := c.Get("user_id").(string)

	group, err := h.store.GetGroup(c.Request().Context(), groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	if group.IsMember(userID) {
		return echo.NewHTTPError(http.StatusBadRequest, "already a member of this group")
	}

	group.Members = append(group.Members, userID)
	if err := h.store.SaveGroup(c.Request().Context(), group); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to join group")
	}

	return c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) LeaveGroup(c echo.Context) error {
	groupID := c.Param("id")
	userID := c.Get("user_id").(string)

	group, err := h.store.GetGroup(c.Request().Context(), groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	if !group.IsMember(userID) {
		return echo.NewHTTPError(http.StatusBadRequest, "not a member of this group")
	}

	for i, member := range group.Members {
		if member == userID {
			group.Members = append(group.Members[:i], group.Members[i+1:]...)
			break
		}
	}

	if err := h.store.SaveGroup(c.Request().Context(), group); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to leave group")
	}

	return c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) RemoveMember(c echo.Context) error {
	groupID := c.Param("id")
	memberID := c.Param("memberID")
	userID := c.Get("user_id").(string)

	group, err := h.store.GetGroup(c.Request().Context(), groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	if group.CreatedBy != userID {
		return echo.NewHTTPError(http.StatusForbidden, "only group creator can remove members")
	}

	if memberID == group.CreatedBy {
		return echo.NewHTTPError(http.StatusForbidden, "cannot remove group creator")
	}

	if !group.IsMember(memberID) {
		return echo.NewHTTPError(http.StatusBadRequest, "user is not a member of this group")
	}
	oldMembers := append([]string{}, group.Members...)
	group.RemoveMember(memberID)

	if err := h.store.UpdateGroupMembers(c.Request().Context(), group, oldMembers); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove member from group")
	}

	return c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) DeleteGroup(c echo.Context) error {
	groupID := c.Param("id")
	userID := c.Get("user_id").(string)

	group, err := h.store.GetGroup(c.Request().Context(), groupID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}

	if group.CreatedBy != userID {
		return echo.NewHTTPError(http.StatusForbidden, "only group creator can delete the group")
	}

	if err := h.store.DeleteGroup(c.Request().Context(), group); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete group")
	}

	return c.NoContent(http.StatusNoContent)
}

package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bm-197/go-chat/internal/api/middleware"
	"github.com/bm-197/go-chat/internal/models"
	"github.com/bm-197/go-chat/internal/store"
)

type UserHandler struct {
	store     *store.RedisStore
	jwtSecret string
}

func NewUserHandler(store *store.RedisStore, jwtSecret string) *UserHandler {
	return &UserHandler{
		store:     store,
		jwtSecret: jwtSecret,
	}
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

func (h *UserHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := models.NewUser(req.Username, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	if err := h.store.SaveUser(c.Request().Context(), user); err != nil {
		if err.Error() == "username already exists" {
			return echo.NewHTTPError(http.StatusConflict, "username already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save user")
	}

	return c.JSON(http.StatusCreated, RegisterResponse{
		UserID:   user.ID,
		Username: user.Username,
	})
}

func (h *UserHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.store.GetUserByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	if !user.ValidatePassword(req.Password) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	token, err := middleware.GenerateToken(user.ID, user.Username, h.jwtSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
	})
}

func (h *UserHandler) GetProfile(c echo.Context) error {
	userID := c.Get("user_id").(string)
	user, err := h.store.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	return c.JSON(http.StatusOK, user)
}

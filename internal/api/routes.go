package api

import (
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/bm-197/go-chat/internal/api/handlers"
	"github.com/bm-197/go-chat/internal/api/middleware"
	"github.com/bm-197/go-chat/internal/store"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func RegisterHandlers(e *echo.Echo, store *store.RedisStore) {
	e.Validator = &CustomValidator{validator: validator.New()}

	userHandler := handlers.NewUserHandler(store, os.Getenv("JWT_SECRET"))
	groupHandler := handlers.NewGroupHandler(store)
	messageHandler := handlers.NewMessageHandler(store)
	wsHandler := handlers.NewWebSocketHandler(store)

	jwtMiddleware := middleware.AuthMiddleware(middleware.JWTConfig{
		SecretKey: os.Getenv("JWT_SECRET"),
	})

	// Public routes
	e.POST("/api/register", userHandler.Register)
	e.POST("/api/login", userHandler.Login)

	// Protected routes
	api := e.Group("/api", jwtMiddleware)

	// User routes
	api.GET("/profile", userHandler.GetProfile)

	// Group routes
	api.POST("/groups", groupHandler.CreateGroup)
	api.GET("/groups", groupHandler.ListGroups)
	api.GET("/groups/:id", groupHandler.GetGroup)
	api.POST("/groups/:id/join", groupHandler.JoinGroup)
	api.POST("/groups/:id/leave", groupHandler.LeaveGroup)
	api.DELETE("/groups/:id/members/:memberID", groupHandler.RemoveMember)
	api.DELETE("/groups/:id", groupHandler.DeleteGroup)

	// Message routes
	api.POST("/messages", messageHandler.SendMessage)
	api.GET("/messages/private/:userID", messageHandler.GetPrivateMessages)
	api.GET("/messages/group/:groupID", messageHandler.GetGroupMessages)
	api.GET("/messages/broadcast", messageHandler.GetBroadcastMessages)

	// WebSocket route
	api.GET("/ws", wsHandler.HandleWebSocket)
}

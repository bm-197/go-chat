package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/bm-197/go-chat/internal/api"
	"github.com/bm-197/go-chat/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	redisStore, err := store.NewRedisStore(
		os.Getenv("REDIS_HOST"),
		os.Getenv("REDIS_PORT"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Register all routes
	api.RegisterHandlers(e, redisStore)

	// Start server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "5000"
	}
	e.Logger.Fatal(e.Start(":" + port))
}

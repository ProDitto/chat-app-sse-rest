package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"

	"sse-chat/internal/config"
	"sse-chat/internal/repository"
	transportHTTP "sse-chat/internal/transport/http"
	"sse-chat/internal/usecase"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Override with environment variables if they exist
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		cfg.Server.Port = serverPort
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		slog.Error("could not connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("successfully connected to Redis")

	// Initialize repositories
	userRepo := repository.NewRedisUserRepository(redisClient)
	eventRepo := repository.NewRedisEventRepository(redisClient)

	// Initialize use cases
	userUsecase := usecase.NewUserUsecase(userRepo, cfg.App.UserInactiveTimeout, cfg.App.UsernameCooldown)
	eventUsecase := usecase.NewEventUsecase(eventRepo, userRepo)

	// Start background cleanup job
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				slog.Info("running inactive user cleanup...")
				cleanedUsers, err := userUsecase.CleanupInactiveUsers(ctx) // Use main ctx
				if err != nil {
					slog.Error("error during inactive user cleanup", "error", err)
				}
				if len(cleanedUsers) > 0 {
					slog.Info("cleaned up inactive users", "count", len(cleanedUsers))
				}
			case <-ctx.Done(): // Listen for main context cancellation
				slog.Info("inactive user cleanup routine stopped.")
				return
			}
		}
	}()

	// Initialize HTTP handler and router
	handler := transportHTTP.NewHandler(userUsecase, eventUsecase, eventRepo, redisClient)
	router := handler.InitRoutes()

	// Setup and start server
	server := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: router,
	}

	go func() {
		slog.Info("starting server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server exiting")
}


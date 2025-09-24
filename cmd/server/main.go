package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"

	"sse-chat/internal/config"
	"sse-chat/internal/repository"
	"sse-chat/internal/transport/http/handler"
	"sse-chat/internal/usecase"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Override with environment variables if they exist
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		cfg.Server.Port = serverPort
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("could not connect to redis: %v", err)
	}

	userRepo := repository.NewRedisUserRepository(redisClient)
	userUsecase := usecase.NewUserUsecase(userRepo, cfg.App.UserInactiveTimeout, cfg.App.UsernameCooldown)

	// Start background cleanup job
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cleanedUsers, err := userUsecase.CleanupInactiveUsers(ctx)
				if err != nil {
					log.Printf("error during inactive user cleanup: %v", err)
				}
				if len(cleanedUsers) > 0 {
					log.Printf("cleaned up %d inactive users", len(cleanedUsers))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	httpHandler := handler.NewHandler(userUsecase)
	router := httpHandler.InitRoutes()

	server := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: router,
	}

	go func() {
		log.Printf("server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("server shutdown failed:", err)
	}

	log.Println("server exited properly")
}


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
	"sse-chat/internal/transport/http/handler" // Use original package name
	"sse-chat/internal/usecase"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
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

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

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
				cleanedUsers, err := userUsecase.CleanupInactiveUsers(ctx) // Use main ctx
				if err != nil {
					log.Printf("Error during inactive user cleanup: %v", err)
				}
				if len(cleanedUsers) > 0 {
					log.Printf("Cleaned up %d inactive users", len(cleanedUsers))
				}
			case <-ctx.Done(): // Listen for main context cancellation
				log.Println("Inactive user cleanup routine stopped.")
				return
			}
		}
	}()

	// Initialize HTTP handler and router
	httpHandler := handler.NewHandler(userUsecase, eventUsecase) // Pass both usecases
	router := httpHandler.InitRoutes()

	// Setup and start server
	server := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: router,
	}

	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

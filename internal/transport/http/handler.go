package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v8"
	mw "sse-chat/internal/transport/http/middleware"
	"sse-chat/internal/usecase"
)

type Handler struct {
	userUsecase  usecase.UserUsecase
	eventUsecase usecase.EventUsecase
	eventRepo    usecase.EventRepository
	redisClient  *redis.Client
}

func NewHandler(userUsecase usecase.UserUsecase, eventUsecase usecase.EventUsecase, eventRepo usecase.EventRepository, redisClient *redis.Client) *Handler {
	return &Handler{
		userUsecase:  userUsecase,
		eventUsecase: eventUsecase,
		eventRepo:    eventRepo,
		redisClient:  redisClient,
	}
}

func (h *Handler) InitRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.Logger) // Using custom logger middleware
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/ping")) // Changed heartbeat path

	// API routes
	r.Route("/api", func(r chi.Router) {
		userHandler := NewUserHandler(h.userUsecase)
		eventHandler := NewEventHandler(h.eventUsecase)
		sseHandler := NewSSEHandler(h.userUsecase, h.eventRepo)

		// Apply rate limiters to specific endpoints
		r.Group(func(r chi.Router) {
			// Join attempts: 3/min/IP
			r.Use(mw.RateLimiter(h.redisClient, "ratelimit:join", 3, 1*time.Minute))
			r.Post("/join", userHandler.Join)
		})

		r.Group(func(r chi.Router) {
			// Username changes: 1 per 30s
			r.Use(mw.RateLimiter(h.redisClient, "ratelimit:username_change", 1, 30*time.Second))
			r.Post("/change-username", userHandler.ChangeUsername)
		})

		r.Group(func(r chi.Router) {
			// Messages: 5 per 5s
			r.Use(mw.RateLimiter(h.redisClient, "ratelimit:message", 5, 5*time.Second))
			r.Post("/message", eventHandler.SendMessage)
		})

		// Other routes without specific rate limits
		r.Post("/typing", eventHandler.SendTypingStatus)
		r.Get("/snapshot", eventHandler.GetSnapshot)
		r.Get("/sse", sseHandler.ServeHTTP)
	})

	// Serve static files
	fs := http.FileServer(http.Dir("./public"))
	r.Handle("/*", fs)

	return r
}

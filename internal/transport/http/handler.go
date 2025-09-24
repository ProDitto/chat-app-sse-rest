package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"sse-chat/internal/usecase"
)

type Handler struct {
	userUsecase  usecase.UserUsecase
	eventUsecase usecase.EventUsecase
	eventRepo    usecase.EventRepository
}

func NewHandler(userUsecase usecase.UserUsecase, eventUsecase usecase.EventUsecase, eventRepo usecase.EventRepository) *Handler {
	return &Handler{
		userUsecase:  userUsecase,
		eventUsecase: eventUsecase,
		eventRepo:    eventRepo,
	}
}

func (h *Handler) InitRoutes() *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Use(middleware.Heartbeat("/health"))

	userHandler := NewUserHandler(h.userUsecase)
	eventHandler := NewEventHandler(h.eventUsecase)
	sseHandler := NewSSEHandler(h.userUsecase, h.eventRepo)

	router.Route("/api", func(r chi.Router) {
		r.Post("/join", userHandler.Join)
		r.Post("/change-username", userHandler.ChangeUsername)
		r.Post("/message", eventHandler.SendMessage)
		r.Post("/typing", eventHandler.SendTypingStatus)
		r.Get("/snapshot", eventHandler.GetSnapshot)
		r.Get("/sse", sseHandler.ServeHTTP)
	})

	// Serve static files
	fs := http.FileServer(http.Dir("./public"))
	router.Handle("/*", fs)

	return router
}


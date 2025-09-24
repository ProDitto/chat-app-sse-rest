package handler

import (
	"net/http"
	"sse-chat/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	userUsecase  usecase.UserUsecase
	eventUsecase usecase.EventUsecase
}

func NewHandler(userUsecase usecase.UserUsecase, eventUsecase usecase.EventUsecase) *Handler {
	return &Handler{
		userUsecase:  userUsecase,
		eventUsecase: eventUsecase,
	}
}

func (h *Handler) InitRoutes() *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Heartbeat("/ping")) // Keep original heartbeat

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	userHandler := NewUserHandler(h.userUsecase)
	eventHandler := NewEventHandler(h.eventUsecase)

	router.Route("/api", func(r chi.Router) {
		r.Post("/join", userHandler.Join)
		r.Post("/change-username", userHandler.ChangeUsername)
		r.Post("/message", eventHandler.SendMessage)
		r.Post("/typing", eventHandler.SendTypingStatus)
	})

	// Serve static files
	fs := http.FileServer(http.Dir("./public"))
	router.Handle("/*", fs)

	return router
}


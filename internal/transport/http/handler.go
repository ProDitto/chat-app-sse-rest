package handler

import (
	"net/http"
	"sse-chat/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	userUsecase usecase.UserUsecase
}

func NewHandler(userUsecase usecase.UserUsecase) *Handler {
	return &Handler{
		userUsecase: userUsecase,
	}
}

func (h *Handler) InitRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/ping"))

	r.Route("/api", func(r chi.Router) {
		userHandler := NewUserHandler(h.userUsecase)
		r.Post("/join", userHandler.Join)
		r.Post("/change-username", userHandler.ChangeUsername)
	})

	// Serve static files
	fs := http.FileServer(http.Dir("./public"))
	r.Handle("/*", fs)

	return r
}


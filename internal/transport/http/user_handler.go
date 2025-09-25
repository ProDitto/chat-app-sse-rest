package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"sse-chat/internal/usecase"
)

type UserHandler struct {
	usecase usecase.UserUsecase
}

func NewUserHandler(uc usecase.UserUsecase) *UserHandler {
	return &UserHandler{usecase: uc}
}

type joinRequest struct {
	DesiredUsername string `json:"desiredUsername"`
}

type joinResponse struct {
	UserID      string      `json:"userId"`
	Username    string      `json:"username"`
	ActiveUsers interface{} `json:"activeUsers"`
}

func (h *UserHandler) Join(w http.ResponseWriter, r *http.Request) {
	var req joinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, activeUsers, err := h.usecase.Join(r.Context(), req.DesiredUsername)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidUsername) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to join chat", http.StatusInternalServerError)
		return
	}

	resp := joinResponse{
		UserID:      user.UserID,
		Username:    user.Username,
		ActiveUsers: activeUsers,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

type changeUsernameRequest struct {
	UserID      string `json:"userId"`
	NewUsername string `json:"newUsername"`
}

func (h *UserHandler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	var req changeUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.usecase.ChangeUsername(r.Context(), req.UserID, req.NewUsername)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUsernameTaken),
			errors.Is(err, usecase.ErrUsernameOnCooldown),
			errors.Is(err, usecase.ErrInvalidUsername):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, usecase.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "Failed to change username", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

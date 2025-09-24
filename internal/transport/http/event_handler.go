package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"sse-chat/internal/domain"
	"sse-chat/internal/usecase"
)

type EventHandler struct {
	uc usecase.EventUsecase
}

func NewEventHandler(uc usecase.EventUsecase) *EventHandler {
	return &EventHandler{uc: uc}
}

type sendMessageRequest struct {
	FromUserID string `json:"fromUserId"`
	ToUserID   string `json:"toUserId"`
	Text       string `json:"text"`
}

func (h *EventHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.uc.SendMessage(r.Context(), req.FromUserID, req.ToUserID, req.Text)
	if err != nil {
		if errors.Is(err, usecase.ErrMessageTooLong) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

type sendTypingRequest struct {
	UserID   string `json:"userId"`
	ToUserID string `json:"toUserId"`
	Typing   bool   `json:"typing"`
}

func (h *EventHandler) SendTypingStatus(w http.ResponseWriter, r *http.Request) {
	var req sendTypingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.uc.BroadcastTyping(r.Context(), req.UserID, req.ToUserID, req.Typing)
	if err != nil {
		http.Error(w, "Failed to send typing status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

type snapshotResponse struct {
	Events      []*domain.Event `json:"events"`
	ActiveUsers []*domain.User  `json:"activeUsers"`
}

func (h *EventHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	lastEventID := r.URL.Query().Get("lastEventId")

	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	events, users, err := h.uc.GetSnapshot(r.Context(), userID, lastEventID)
	if err != nil {
		http.Error(w, "Failed to get snapshot", http.StatusInternalServerError)
		return
	}

	resp := snapshotResponse{
		Events:      events,
		ActiveUsers: users,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}


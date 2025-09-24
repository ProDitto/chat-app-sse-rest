package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sse-chat/internal/usecase"
)

type EventHandler struct {
	eventUsecase usecase.EventUsecase
}

func NewEventHandler(uc usecase.EventUsecase) *EventHandler {
	return &EventHandler{eventUsecase: uc}
}

type sendMessageRequest struct {
	FromUserID string `json:"fromUserId"`
	ToUserID   string `json:"toUserId"`
	Text       string `json:"messageText"`
}

type sendTypingRequest struct {
	UserID   string `json:"userId"`
	ToUserID string `json:"toUserId"`
	Typing   bool   `json:"typing"`
}

func (h *EventHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.eventUsecase.SendMessage(r.Context(), req.FromUserID, req.ToUserID, req.Text)
	if err != nil {
		if errors.Is(err, usecase.ErrMessageTooLong) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Error sending message: %v", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *EventHandler) SendTypingStatus(w http.ResponseWriter, r *http.Request) {
	var req sendTypingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.eventUsecase.BroadcastTyping(r.Context(), req.UserID, req.ToUserID, req.Typing)
	if err != nil {
		log.Printf("Error broadcasting typing status: %v", err)
		http.Error(w, "Failed to send typing status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}


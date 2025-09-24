package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sse-chat/internal/domain"
	"sse-chat/internal/usecase"
	"time"
)

const (
	heartbeatInterval = 20 * time.Second
	// How long to block on Redis before checking for context cancellation or sending a heartbeat
	redisBlockTimeout = 5 * time.Second
)

type SSEHandler struct {
	userUsecase usecase.UserUsecase
	eventRepo   usecase.EventRepository
}

func NewSSEHandler(userUC usecase.UserUsecase, eventRepo usecase.EventRepository) *SSEHandler {
	return &SSEHandler{
		userUsecase: userUC,
		eventRepo:   eventRepo,
	}
}

func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID == "" {
		lastEventID = r.URL.Query().Get("lastEventId")
	}

	if userID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	log.Printf("SSE connection established for user %s", userID)
	defer log.Printf("SSE connection closed for user %s", userID)

	// 1. Initial catch-up for any missed events while disconnected.
	events, err := h.eventRepo.GetEventsForUserAfter(ctx, userID, lastEventID)
	if err != nil {
		log.Printf("Error getting initial events for user %s: %v", userID, err)
		http.Error(w, "Failed to retrieve events", http.StatusInternalServerError)
		return
	}
	lastEventID = h.sendEvents(ctx, w, flusher, userID, events, lastEventID)

	// 2. Start heartbeat ticker
	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	// 3. Main event loop
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if err := h.sendHeartbeat(w, flusher); err != nil {
				log.Printf("Error sending heartbeat to user %s: %v", userID, err)
				return
			}
		default:
			// Block and wait for new events
			newEvents, err := h.eventRepo.GetEventsForUserBlocking(ctx, userID, lastEventID, redisBlockTimeout)
			if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				log.Printf("Error getting blocking events for user %s: %v", userID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if len(newEvents) > 0 {
				lastEventID = h.sendEvents(ctx, w, flusher, userID, newEvents, lastEventID)
			}
		}
	}
}

func (h *SSEHandler) sendEvents(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, userID string, events []*domain.Event, lastEventID string) string {
	if len(events) == 0 {
		return lastEventID
	}

	newLastEventID := lastEventID
	for _, event := range events {
		if err := h.sendEvent(w, flusher, event); err != nil {
			log.Printf("Error sending event %s to user %s: %v", event.EventID, userID, err)
			return newLastEventID
		}
		newLastEventID = event.EventID
	}

	if err := h.userUsecase.UpdateLastDeliveredEventID(ctx, userID, newLastEventID); err != nil {
		log.Printf("Error updating last delivered event ID for user %s: %v", userID, err)
	}

	return newLastEventID
}

func (h *SSEHandler) sendHeartbeat(w http.ResponseWriter, flusher http.Flusher) error {
	fmt.Fprintf(w, "event: %s\n", domain.Heartbeat)
	fmt.Fprintf(w, "data: {}\n\n")
	flusher.Flush()
	return nil
}

func (h *SSEHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event *domain.Event) error {
	if event.EventID != "" {
		fmt.Fprintf(w, "id: %s\n", event.EventID)
	}
	if event.Type != "" {
		fmt.Fprintf(w, "event: %s\n", event.Type)
	}

	var dataBytes []byte
	var err error

	if event.Payload != nil {
		dataBytes, err = json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal event payload: %w", err)
		}
	} else {
		dataBytes = []byte("{}")
	}

	fmt.Fprintf(w, "data: %s\n\n", string(dataBytes))
	flusher.Flush()
	return nil
}


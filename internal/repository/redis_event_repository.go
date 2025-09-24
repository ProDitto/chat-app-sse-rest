package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sse-chat/internal/domain"
	"sse-chat/internal/usecase"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	broadcastChannel = "sse-chat:broadcast"
	maxStreamLen     = 1000 // Max events per user stream
)

type redisEventRepository struct {
	client *redis.Client
}

func NewRedisEventRepository(client *redis.Client) usecase.EventRepository {
	return &redisEventRepository{client: client}
}

func userStreamKey(userID string) string {
	return fmt.Sprintf("user:%s:events", userID)
}

func (r *redisEventRepository) PublishEventToUser(ctx context.Context, userID string, event *domain.Event) error {
	values, err := eventToMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to map: %w", err)
	}

	// The event ID from the domain model is used as the stream entry ID.
	// Redis Streams can generate IDs, but we want deterministic IDs.
	// However, Redis Stream IDs must be in format "timestamp-sequence".
	// We will let Redis generate the ID and store our domain EventID in the message body.
	// The generated ID will be returned and can be used for `lastDeliveredEventId`.
	cmd := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: userStreamKey(userID),
		MaxLen: maxStreamLen,
		Approx: true, // Use ~ for approximate trimming
		Values: values,
	})

	eventID, err := cmd.Result()
	if err != nil {
		return fmt.Errorf("failed to add event to stream for user %s: %w", userID, err)
	}

	// Update the event with the actual ID from Redis Stream
	event.EventID = eventID

	return nil
}

func (r *redisEventRepository) PublishEventToAll(ctx context.Context, event *domain.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast event: %w", err)
	}

	return r.client.Publish(ctx, broadcastChannel, payload).Err()
}

func (r *redisEventRepository) GetEventsForUserAfter(ctx context.Context, userID, lastEventID string) ([]*domain.Event, error) {
	start := "-"
	if lastEventID != "" && lastEventID != "0" {
		start = fmt.Sprintf("(%s", lastEventID) // Exclusive start
	}

	messages, err := r.client.XRange(ctx, userStreamKey(userID), start, "+").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get event range for user %s: %w", userID, err)
	}

	events := make([]*domain.Event, 0, len(messages))
	for _, msg := range messages {
		event, err := mapToEvent(msg.ID, msg.Values)
		if err != nil {
			log.Printf("Warning: failed to unmarshal event from stream for user %s: %v", userID, err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *redisEventRepository) SubscribeToBroadcasts(ctx context.Context) *redis.PubSub {
	return r.client.Subscribe(ctx, broadcastChannel)
}

func eventToMap(event *domain.Event) (map[string]interface{}, error) {
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"timestamp":  event.Timestamp.Format(time.RFC3339Nano),
		"type":       string(event.Type),
		"fromUserID": event.FromUserID,
		"toUserID":   event.ToUserID,
		"payload":    string(payloadBytes),
	}, nil
}

func mapToEvent(eventID string, data map[string]interface{}) (*domain.Event, error) {
	event := &domain.Event{
		EventID: eventID,
	}

	if tsStr, ok := data["timestamp"].(string); ok {
		ts, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp format: %w", err)
		}
		event.Timestamp = ts
	}

	if typeStr, ok := data["type"].(string); ok {
		event.Type = domain.EventType(typeStr)
	}

	if from, ok := data["fromUserID"].(string); ok {
		event.FromUserID = from
	}

	if to, ok := data["toUserID"].(string); ok {
		event.ToUserID = to
	}

	if payloadStr, ok := data["payload"].(string); ok {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		event.Payload = payload
	}

	return event, nil
}


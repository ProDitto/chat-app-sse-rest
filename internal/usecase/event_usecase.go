package usecase

import (
	"context"
	"errors"
	"fmt"
	"sse-chat/internal/domain"
	"time"
)

var (
	ErrMessageTooLong = errors.New("message text exceeds maximum length")
)

const maxMessageLength = 500

type eventUsecase struct {
	eventRepo EventRepository
	userRepo  UserRepository
}

func NewEventUsecase(eventRepo EventRepository, userRepo UserRepository) EventUsecase {
	return &eventUsecase{
		eventRepo: eventRepo,
		userRepo:  userRepo,
	}
}

func (uc *eventUsecase) SendMessage(ctx context.Context, fromUserID, toUserID, text string) (*domain.Event, error) {
	if len(text) > maxMessageLength {
		return nil, ErrMessageTooLong
	}
	if text == "" {
		return nil, errors.New("message text cannot be empty")
	}

	event := &domain.Event{
		Timestamp:  time.Now().UTC(),
		Type:       domain.Message,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Payload: map[string]interface{}{
			"fromUserId": fromUserID,
			"toUserId":   toUserID,
			"text":       text,
		},
	}

	// Publish to recipient's stream
	if err := uc.eventRepo.PublishEventToUser(ctx, toUserID, event); err != nil {
		return nil, fmt.Errorf("failed to publish message to recipient: %w", err)
	}

	// Publish to sender's stream (for history)
	if err := uc.eventRepo.PublishEventToUser(ctx, fromUserID, event); err != nil {
		return nil, fmt.Errorf("failed to publish message to sender: %w", err)
	}

	// Update sender's activity
	user, err := uc.userRepo.FindByID(ctx, fromUserID)
	if err == nil && user != nil {
		user.LastActive = time.Now().UTC()
		uc.userRepo.Update(ctx, user)
	}

	return event, nil
}

func (uc *eventUsecase) BroadcastTyping(ctx context.Context, fromUserID, toUserID string, isTyping bool) error {
	eventType := domain.TypingStop
	if isTyping {
		eventType = domain.TypingStart
	}

	event := &domain.Event{
		Timestamp:  time.Now().UTC(),
		Type:       eventType,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Payload: map[string]interface{}{
			"fromUserId": fromUserID,
			"toUserId":   toUserID,
		},
	}

	if err := uc.eventRepo.PublishEventToUser(ctx, toUserID, event); err != nil {
		return fmt.Errorf("failed to publish typing event: %w", err)
	}

	// Update sender's activity
	user, err := uc.userRepo.FindByID(ctx, fromUserID)
	if err == nil && user != nil {
		user.LastActive = time.Now().UTC()
		uc.userRepo.Update(ctx, user)
	}

	return nil
}

func (uc *eventUsecase) GetSnapshot(ctx context.Context, userID, lastEventID string) ([]*domain.Event, []*domain.User, error) {
	events, err := uc.eventRepo.GetEventsForUserAfter(ctx, userID, lastEventID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get events for user: %w", err)
	}

	activeUsers, err := uc.userRepo.GetActiveUsers(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active users: %w", err)
	}

	return events, activeUsers, nil
}


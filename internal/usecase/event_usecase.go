package usecase

import (
	"context"
	"errors"
	"sse-chat/internal/domain"
	"sse-chat/pkg/utils"
	"time"
)

var (
	ErrMessageTooLong = errors.New("message text is too long")
)

const maxMessageLength = 500

type eventUsecase struct {
	eventRepo usecase.EventRepository
	userRepo  usecase.UserRepository
}

func NewEventUsecase(eventRepo usecase.EventRepository, userRepo usecase.UserRepository) usecase.EventUsecase {
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
		EventID:    utils.GenerateID(), // This is a placeholder; Redis will generate the final stream ID
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
		return nil, err
	}

	// Publish to sender's stream for sync across their own clients
	if err := uc.eventRepo.PublishEventToUser(ctx, fromUserID, event); err != nil {
		return nil, err
	}

	// Update sender's activity
	if err := uc.userRepo.Update(ctx, &domain.User{UserID: fromUserID, LastActive: time.Now().UTC()}); err != nil {
		// Log error but don't fail the message sending
	}

	return event, nil
}

func (uc *eventUsecase) BroadcastTyping(ctx context.Context, fromUserID, toUserID string, isTyping bool) error {
	eventType := domain.TypingStart
	if !isTyping {
		eventType = domain.TypingStop
	}

	event := &domain.Event{
		EventID:    utils.GenerateID(),
		Timestamp:  time.Now().UTC(),
		Type:       eventType,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Payload: map[string]interface{}{
			"fromUserId": fromUserID,
			"toUserId":   toUserID,
		},
	}

	// Only publish to the recipient
	if err := uc.eventRepo.PublishEventToUser(ctx, toUserID, event); err != nil {
		return err
	}

	// Update sender's activity
	if err := uc.userRepo.Update(ctx, &domain.User{UserID: fromUserID, LastActive: time.Now().UTC()}); err != nil {
		// Log error but don't fail the typing event
	}

	return nil
}


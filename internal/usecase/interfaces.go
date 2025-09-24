package usecase

import (
	"context"
	"sse-chat/internal/domain"
	"time"

	"github.com/go-redis/redis/v8"
)

// UserRepository defines the contract for data access operations related to users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, userID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, userID string) error
	GetActiveUsers(ctx context.Context) ([]*domain.User, error)
	FindInactiveUsers(ctx context.Context, timeout time.Duration) ([]string, error)
	IsUsernameTaken(ctx context.Context, username string) (bool, error)
	ReleaseUsername(ctx context.Context, username string, cooldown time.Duration) error
	IsUsernameOnCooldown(ctx context.Context, username string) (bool, error)
}

// UserUsecase defines the contract for business logic operations related to users.
type UserUsecase interface {
	Join(ctx context.Context, desiredUsername string) (*domain.User, []*domain.User, error)
	ChangeUsername(ctx context.Context, userID, newUsername string) (string, error)
	UpdateActivity(ctx context.Context, userID string) error
	Disconnect(ctx context.Context, userID string) (*domain.User, error)
	CleanupInactiveUsers(ctx context.Context) ([]*domain.User, error)
}

// EventRepository defines the contract for event persistence and broadcasting.
type EventRepository interface {
	// PublishEventToUser adds an event to a specific user's stream.
	PublishEventToUser(ctx context.Context, userID string, event *domain.Event) error
	// PublishEventToAll broadcasts an event to all server instances.
	PublishEventToAll(ctx context.Context, event *domain.Event) error
	// GetEventsForUserAfter retrieves events for a user from their stream after a given event ID.
	GetEventsForUserAfter(ctx context.Context, userID, lastEventID string) ([]*domain.Event, error)
	// SubscribeToBroadcasts subscribes to the global event broadcast channel.
	SubscribeToBroadcasts(ctx context.Context) *redis.PubSub
}

// EventUsecase defines the contract for business logic related to events.
type EventUsecase interface {
	SendMessage(ctx context.Context, fromUserID, toUserID, text string) (*domain.Event, error)
	BroadcastTyping(ctx context.Context, fromUserID, toUserID string, isTyping bool) error
}


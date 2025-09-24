package usecase

import (
	"context"
	"sse-chat/internal/domain"
	"time"
)

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

type UserUsecase interface {
	Join(ctx context.Context, desiredUsername string) (*domain.User, []*domain.User, error)
	ChangeUsername(ctx context.Context, userID, newUsername string) (oldUsername string, err error)
	UpdateActivity(ctx context.Context, userID string) error
	Disconnect(ctx context.Context, userID string) (*domain.User, error)
	CleanupInactiveUsers(ctx context.Context) ([]*domain.User, error)
}


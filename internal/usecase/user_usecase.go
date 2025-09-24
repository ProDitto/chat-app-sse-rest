package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sse-chat/internal/domain"
	"sse-chat/pkg/utils"
	"strings"
	"time"
)

var (
	ErrUsernameTaken      = errors.New("username is already taken")
	ErrUsernameOnCooldown = errors.New("username is on cooldown")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidUsername    = errors.New("invalid username")
)

type userUsecase struct {
	userRepo            UserRepository
	userInactiveTimeout time.Duration
	usernameCooldown    time.Duration
}

func NewUserUsecase(userRepo UserRepository, inactiveTimeout, cooldown time.Duration) UserUsecase {
	return &userUsecase{
		userRepo:            userRepo,
		userInactiveTimeout: inactiveTimeout,
		usernameCooldown:    cooldown,
	}
}

func (uc *userUsecase) Join(ctx context.Context, desiredUsername string) (*domain.User, []*domain.User, error) {
	username := strings.TrimSpace(desiredUsername)
	if username == "" {
		return nil, nil, ErrInvalidUsername
	}

	isTaken, err := uc.userRepo.IsUsernameTaken(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check username: %w", err)
	}
	if isTaken {
		username = fmt.Sprintf("%s%d", username, rand.Intn(1000))
	}

	user := &domain.User{
		UserID:     utils.GenerateID(),
		Username:   username,
		LastActive: time.Now().UTC(),
		IsOnline:   true,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	activeUsers, err := uc.userRepo.GetActiveUsers(ctx)
	if err != nil {
		// Log the error but don't fail the join operation
		fmt.Printf("failed to get active users after join: %v", err)
	}

	return user, activeUsers, nil
}

func (uc *userUsecase) ChangeUsername(ctx context.Context, userID, newUsername string) (string, error) {
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return "", ErrInvalidUsername
	}

	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	oldUsername := user.Username
	if oldUsername == newUsername {
		return oldUsername, nil // No change
	}

	isTaken, err := uc.userRepo.IsUsernameTaken(ctx, newUsername)
	if err != nil {
		return "", fmt.Errorf("failed to check username availability: %w", err)
	}
	if isTaken {
		return "", ErrUsernameTaken
	}

	isOnCooldown, err := uc.userRepo.IsUsernameOnCooldown(ctx, newUsername)
	if err != nil {
		return "", fmt.Errorf("failed to check username cooldown: %w", err)
	}
	if isOnCooldown {
		return "", ErrUsernameOnCooldown
	}

	user.Username = newUsername
	user.LastActive = time.Now().UTC()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	if err := uc.userRepo.ReleaseUsername(ctx, oldUsername, uc.usernameCooldown); err != nil {
		// Log this error, but don't fail the operation as the username change was successful
		fmt.Printf("failed to release old username '%s': %v", oldUsername, err)
	}

	return oldUsername, nil
}

func (uc *userUsecase) UpdateActivity(ctx context.Context, userID string) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	user.LastActive = time.Now().UTC()
	return uc.userRepo.Update(ctx, user)
}

func (uc *userUsecase) Disconnect(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	user.IsOnline = false
	err = uc.userRepo.Update(ctx, user)
	return user, err
}

func (uc *userUsecase) CleanupInactiveUsers(ctx context.Context) ([]*domain.User, error) {
	inactiveIDs, err := uc.userRepo.FindInactiveUsers(ctx, uc.userInactiveTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to find inactive users: %w", err)
	}

	cleanedUsers := make([]*domain.User, 0, len(inactiveIDs))
	for _, userID := range inactiveIDs {
		user, err := uc.userRepo.FindByID(ctx, userID)
		if err != nil {
			continue // User might have been deleted already
		}

		if err := uc.userRepo.Delete(ctx, userID); err != nil {
			fmt.Printf("failed to delete inactive user %s: %v", userID, err)
			continue
		}

		if err := uc.userRepo.ReleaseUsername(ctx, user.Username, uc.usernameCooldown); err != nil {
			fmt.Printf("failed to release username for inactive user %s: %v", userID, err)
		}
		cleanedUsers = append(cleanedUsers, user)
	}

	return cleanedUsers, nil
}

func (uc *userUsecase) UpdateLastDeliveredEventID(ctx context.Context, userID, eventID string) error {
	return uc.userRepo.UpdateLastDeliveredEventID(ctx, userID, eventID)
}

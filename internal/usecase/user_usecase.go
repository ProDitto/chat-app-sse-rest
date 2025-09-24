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
	ErrUsernameTaken      = errors.New("username is taken")
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
		return nil, nil, err
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
		return nil, nil, err
	}

	activeUsers, err := uc.userRepo.GetActiveUsers(ctx)
	if err != nil {
		// Log the error but continue, as the user has successfully joined.
		// The client can fetch the user list separately if needed.
		fmt.Printf("error getting active users after join: %v\n", err)
	}

	return user, activeUsers, nil
}

func (uc *userUsecase) ChangeUsername(ctx context.Context, userID, newUsername string) (string, error) {
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return "", ErrInvalidUsername
	}

	onCooldown, err := uc.userRepo.IsUsernameOnCooldown(ctx, newUsername)
	if err != nil {
		return "", err
	}
	if onCooldown {
		return "", ErrUsernameOnCooldown
	}

	isTaken, err := uc.userRepo.IsUsernameTaken(ctx, newUsername)
	if err != nil {
		return "", err
	}
	if isTaken {
		return "", ErrUsernameTaken
	}

	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}

	oldUsername := user.Username
	if oldUsername == newUsername {
		return oldUsername, nil // No change
	}

	user.Username = newUsername
	user.LastActive = time.Now().UTC()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return "", err
	}

	if err := uc.userRepo.ReleaseUsername(ctx, oldUsername, uc.usernameCooldown); err != nil {
		// Log this error, but don't fail the whole operation
		fmt.Printf("failed to release old username %s: %v\n", oldUsername, err)
	}

	return oldUsername, nil
}

func (uc *userUsecase) UpdateActivity(ctx context.Context, userID string) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	user.LastActive = time.Now().UTC()
	return uc.userRepo.Update(ctx, user)
}

func (uc *userUsecase) Disconnect(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.IsOnline = false
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *userUsecase) CleanupInactiveUsers(ctx context.Context) ([]*domain.User, error) {
	inactiveUserIDs, err := uc.userRepo.FindInactiveUsers(ctx, uc.userInactiveTimeout)
	if err != nil {
		return nil, err
	}

	var cleanedUsers []*domain.User
	for _, userID := range inactiveUserIDs {
		user, err := uc.userRepo.FindByID(ctx, userID)
		if err != nil {
			fmt.Printf("error finding user %s for cleanup: %v\n", userID, err)
			continue
		}
		if user == nil {
			continue
		}

		if err := uc.userRepo.Delete(ctx, userID); err != nil {
			fmt.Printf("error deleting user %s during cleanup: %v\n", userID, err)
			continue
		}

		if err := uc.userRepo.ReleaseUsername(ctx, user.Username, uc.usernameCooldown); err != nil {
			fmt.Printf("error releasing username %s during cleanup: %v\n", user.Username, err)
		}
		cleanedUsers = append(cleanedUsers, user)
	}

	return cleanedUsers, nil
}


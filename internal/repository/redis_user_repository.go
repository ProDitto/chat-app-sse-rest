package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"sse-chat/internal/domain"
)

const (
	activeUsersKey = "active_users"
	usernamesKey   = "usernames"
)

func userKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func usernameCooldownKey(username string) string {
	return fmt.Sprintf("cooldown:%s", username)
}

type redisUserRepository struct {
	client *redis.Client
}

func NewRedisUserRepository(client *redis.Client) *redisUserRepository {
	return &redisUserRepository{client: client}
}

func (r *redisUserRepository) Create(ctx context.Context, user *domain.User) error {
	pipe := r.client.TxPipeline()

	pipe.HSet(ctx, userKey(user.UserID), map[string]interface{}{
		"username":             user.Username,
		"lastActive":           user.LastActive.Unix(),
		"isOnline":             user.IsOnline,
		"lastDeliveredEventId": user.LastDeliveredEventID,
	})
	pipe.ZAdd(ctx, activeUsersKey, &redis.Z{
		Score:  float64(user.LastActive.Unix()),
		Member: user.UserID,
	})
	pipe.SAdd(ctx, usernamesKey, user.Username)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) FindByID(ctx context.Context, userID string) (*domain.User, error) {
	data, err := r.client.HGetAll(ctx, userKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, redis.Nil
	}

	user := &domain.User{UserID: userID}
	user.Username = data["username"]
	user.LastDeliveredEventID = data["lastDeliveredEventId"]

	if lastActive, err := strconv.ParseInt(data["lastActive"], 10, 64); err == nil {
		user.LastActive = time.Unix(lastActive, 0)
	}
	if isOnline, err := strconv.ParseBool(data["isOnline"]); err == nil {
		user.IsOnline = isOnline
	}

	return user, nil
}

func (r *redisUserRepository) Update(ctx context.Context, user *domain.User) error {
	pipe := r.client.TxPipeline()

	pipe.HSet(ctx, userKey(user.UserID), map[string]interface{}{
		"username":             user.Username,
		"lastActive":           user.LastActive.Unix(),
		"isOnline":             user.IsOnline,
		"lastDeliveredEventId": user.LastDeliveredEventID,
	})
	pipe.ZAdd(ctx, activeUsersKey, &redis.Z{
		Score:  float64(user.LastActive.Unix()),
		Member: user.UserID,
	})
	pipe.SAdd(ctx, usernamesKey, user.Username)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) Delete(ctx context.Context, userID string) error {
	user, err := r.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, userKey(userID))
	pipe.ZRem(ctx, activeUsersKey, userID)
	if user != nil { // Safeguard in case FindByID returns nil user but no error (e.g., redis.Nil)
		pipe.SRem(ctx, usernamesKey, user.Username)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) GetActiveUsers(ctx context.Context) ([]*domain.User, error) {
	userIDs, err := r.client.ZRange(ctx, activeUsersKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	users := make([]*domain.User, 0, len(userIDs))
	for _, id := range userIDs {
		user, err := r.FindByID(ctx, id)
		if err == redis.Nil {
			continue // User might have been deleted concurrently
		}
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *redisUserRepository) FindInactiveUsers(ctx context.Context, timeout time.Duration) ([]string, error) {
	maxScore := time.Now().Add(-timeout).Unix()
	return r.client.ZRangeByScore(ctx, activeUsersKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(maxScore, 10),
	}).Result()
}

func (r *redisUserRepository) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	return r.client.SIsMember(ctx, usernamesKey, username).Result()
}

func (r *redisUserRepository) ReleaseUsername(ctx context.Context, username string, cooldown time.Duration) error {
	pipe := r.client.TxPipeline()
	pipe.SRem(ctx, usernamesKey, username)
	pipe.Set(ctx, usernameCooldownKey(username), "1", cooldown)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) IsUsernameOnCooldown(ctx context.Context, username string) (bool, error) {
	err := r.client.Get(ctx, usernameCooldownKey(username)).Err()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *redisUserRepository) UpdateLastDeliveredEventID(ctx context.Context, userID, eventID string) error {
	return r.client.HSet(ctx, userKey(userID), "lastDeliveredEventId", eventID).Err()
}


package repository

import (
	"context"
	"encoding/json"
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
	return fmt.Sprintf("username_cooldown:%s", username)
}

type redisUserRepository struct {
	client *redis.Client
}

func NewRedisUserRepository(client *redis.Client) *redisUserRepository {
	return &redisUserRepository{client: client}
}

func (r *redisUserRepository) Create(ctx context.Context, user *domain.User) error {
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, userKey(user.UserID), userData, 0)
	pipe.ZAdd(ctx, activeUsersKey, &redis.Z{Score: float64(user.LastActive.Unix()), Member: user.UserID})
	pipe.SAdd(ctx, usernamesKey, user.Username)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) FindByID(ctx context.Context, userID string) (*domain.User, error) {
	data, err := r.client.Get(ctx, userKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var user domain.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *redisUserRepository) Update(ctx context.Context, user *domain.User) error {
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, userKey(user.UserID), userData, 0)
	pipe.ZAdd(ctx, activeUsersKey, &redis.Z{Score: float64(user.LastActive.Unix()), Member: user.UserID})
	// If username changed, we need to update the usernames set
	pipe.SAdd(ctx, usernamesKey, user.Username)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) Delete(ctx context.Context, userID string) error {
	user, err := r.FindByID(ctx, userID)
	if err != nil || user == nil {
		return err
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, userKey(userID))
	pipe.ZRem(ctx, activeUsersKey, userID)
	pipe.SRem(ctx, usernamesKey, user.Username)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) GetActiveUsers(ctx context.Context) ([]*domain.User, error) {
	userIDs, err := r.client.ZRange(ctx, activeUsersKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	if len(userIDs) == 0 {
		return []*domain.User{}, nil
	}

	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(userIDs))
	for i, id := range userIDs {
		cmds[i] = pipe.Get(ctx, userKey(id))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	users := make([]*domain.User, 0, len(userIDs))
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			continue // User might have been deleted between ZRANGE and GET
		}
		if err != nil {
			return nil, err
		}
		var user domain.User
		if err := json.Unmarshal([]byte(data), &user); err == nil {
			users = append(users, &user)
		}
	}
	return users, nil
}

func (r *redisUserRepository) FindInactiveUsers(ctx context.Context, timeout time.Duration) ([]string, error) {
	maxScore := strconv.FormatInt(time.Now().UTC().Add(-timeout).Unix(), 10)
	return r.client.ZRangeByScore(ctx, activeUsersKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: maxScore,
	}).Result()
}

func (r *redisUserRepository) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	return r.client.SIsMember(ctx, usernamesKey, username).Result()
}

func (r *redisUserRepository) ReleaseUsername(ctx context.Context, username string, cooldown time.Duration) error {
	pipe := r.client.TxPipeline()
	pipe.SRem(ctx, usernamesKey, username)
	pipe.SetEX(ctx, usernameCooldownKey(username), "1", cooldown)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisUserRepository) IsUsernameOnCooldown(ctx context.Context, username string) (bool, error) {
	val, err := r.client.Exists(ctx, usernameCooldownKey(username)).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}


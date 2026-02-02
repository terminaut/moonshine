package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"moonshine/internal/config"
	"moonshine/internal/domain"

	"github.com/redis/go-redis/v9"
)

func New(config *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       0,
	})
}

func Ping(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}

func GetUser(ctx context.Context, rdb *redis.Client, key string) (*domain.User, error) {
	key = "user:" + key
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var user domain.User
	if err := json.Unmarshal([]byte(value), &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func SetUser(ctx context.Context, rdb *redis.Client, key string, user *domain.User) error {
	key = "user:" + key
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	err = rdb.Set(ctx, key, data, 0).Err() // 0 = без TTL, можно добавить TTL
	if err != nil {
		return err
	}

	return nil
}

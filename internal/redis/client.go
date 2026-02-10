package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"moonshine/internal/config"

	"github.com/redis/go-redis/v9"
)

type Cache[T any] interface {
	Get(ctx context.Context, key string) (*T, error)
	Set(ctx context.Context, key string, value *T) error
	Delete(ctx context.Context, key string) error
}

type JSONCache[T any] struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

func New(config *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       0,
	})
}

func NewJSONCache[T any](client *redis.Client, prefix string, ttl time.Duration) *JSONCache[T] {
	return &JSONCache[T]{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

func Ping(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}

func (c *JSONCache[T]) Get(ctx context.Context, key string) (*T, error) {
	value, err := c.client.Get(ctx, c.formatKey(key)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var result T
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("unmarshal cache value: %w", err)
	}

	return &result, nil
}

func (c *JSONCache[T]) Set(ctx context.Context, key string, value *T) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}

	return c.client.Set(ctx, c.formatKey(key), data, c.ttl).Err()
}

func (c *JSONCache[T]) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.formatKey(key)).Err()
}

func (c *JSONCache[T]) formatKey(key string) string {
	return c.prefix + ":" + key
}

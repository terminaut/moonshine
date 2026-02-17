package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func setupTestRedis(t *testing.T) *goredis.Client {
	t.Helper()
	s := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: s.Addr()})
}

func TestJSONCache_GetMiss(t *testing.T) {
	client := setupTestRedis(t)
	cache := NewJSONCache[testItem](client, "test", 5*time.Second)

	result, err := cache.Get(context.Background(), "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestJSONCache_SetAndGet(t *testing.T) {
	client := setupTestRedis(t)
	cache := NewJSONCache[testItem](client, "test", 5*time.Second)
	ctx := context.Background()

	item := &testItem{Name: "sword", Value: 42}
	err := cache.Set(ctx, "item1", item)
	require.NoError(t, err)

	result, err := cache.Get(ctx, "item1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "sword", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestJSONCache_Delete(t *testing.T) {
	client := setupTestRedis(t)
	cache := NewJSONCache[testItem](client, "test", 5*time.Second)
	ctx := context.Background()

	item := &testItem{Name: "shield", Value: 10}
	require.NoError(t, cache.Set(ctx, "item2", item))

	err := cache.Delete(ctx, "item2")
	require.NoError(t, err)

	result, err := cache.Get(ctx, "item2")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestJSONCache_NilClient(t *testing.T) {
	cache := NewJSONCache[testItem](nil, "test", 5*time.Second)
	ctx := context.Background()

	result, err := cache.Get(ctx, "key")
	assert.NoError(t, err)
	assert.Nil(t, result)

	err = cache.Set(ctx, "key", &testItem{Name: "x", Value: 1})
	assert.NoError(t, err)

	err = cache.Delete(ctx, "key")
	assert.NoError(t, err)
}

func TestJSONCache_NilCache(t *testing.T) {
	var cache *JSONCache[testItem]
	ctx := context.Background()

	result, err := cache.Get(ctx, "key")
	assert.NoError(t, err)
	assert.Nil(t, result)

	err = cache.Set(ctx, "key", &testItem{})
	assert.NoError(t, err)

	err = cache.Delete(ctx, "key")
	assert.NoError(t, err)
}

func TestJSONCache_KeyFormat(t *testing.T) {
	client := setupTestRedis(t)
	cache := NewJSONCache[testItem](client, "user", 5*time.Second)
	ctx := context.Background()

	item := &testItem{Name: "test", Value: 1}
	require.NoError(t, cache.Set(ctx, "abc-123", item))

	val, err := client.Get(ctx, "user:abc-123").Result()
	require.NoError(t, err)
	assert.Contains(t, val, "test")
}

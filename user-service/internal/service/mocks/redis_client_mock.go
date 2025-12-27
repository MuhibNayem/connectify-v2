package mocks

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockRedisClient is a mock implementation of redis.UniversalClient
type MockRedisClient struct {
	GetFunc  func(ctx context.Context, key string) *redis.StringCmd
	MGetFunc func(ctx context.Context, keys ...string) *redis.SliceCmd
	SetFunc  func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd

	GetCalls  []string
	MGetCalls [][]string
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	m.GetCalls = append(m.GetCalls, key)
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	cmd := redis.NewStringCmd(ctx)
	cmd.SetErr(redis.Nil)
	return cmd
}

func (m *MockRedisClient) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	m.MGetCalls = append(m.MGetCalls, keys)
	if m.MGetFunc != nil {
		return m.MGetFunc(ctx, keys...)
	}
	cmd := redis.NewSliceCmd(ctx)
	result := make([]interface{}, len(keys))
	cmd.SetVal(result)
	return cmd
}

// Implement other required methods from redis.UniversalClient with no-op
func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, expiration)
	}
	return redis.NewStatusCmd(ctx)
}

func (m *MockRedisClient) Close() error { return nil }
func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusCmd(ctx)
}

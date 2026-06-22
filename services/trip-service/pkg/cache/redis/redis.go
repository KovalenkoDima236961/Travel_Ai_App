package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Unwrap exposes the underlying go-redis client for advanced use.
func (c *RedisClient) Unwrap() *redis.Client { return c.c }

// HealthCheck verifies connectivity with a PING.
func (c *RedisClient) HealthCheck(ctx context.Context) error {
	if err := c.c.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// Close releases the underlying connection pool.
func (c *RedisClient) Close() error {
	return c.c.Close()
}

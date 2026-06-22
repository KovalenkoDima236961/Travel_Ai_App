package redis

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	c *redis.Client
}

func NewClient(ctx context.Context, cfg *Config) (*RedisClient, error) {
	const op = "redis.NewClient"

	client := redis.NewClient(&redis.Options{
		Addr:            net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Username:        cfg.Username,
		Password:        cfg.Password,
		MaxRetries:      cfg.Retries,
		Protocol:        cfg.Protocol,
		DB:              cfg.Database,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
	})

	c := &RedisClient{client}

	// Test connection
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.HealthCheck(testCtx); err != nil {
		return nil, fmt.Errorf("%s: failed to connect Redis: %w", op, err)
	}

	return c, nil
}
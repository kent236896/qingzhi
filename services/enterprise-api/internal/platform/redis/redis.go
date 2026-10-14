// Package redis checks connectivity to Redis. It does not implement cache or queue behavior.
package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/platform/database"
)

var errRedisNotInitialized = errors.New("redis handle is not initialized")

// Client is a Redis connection.
type Client struct {
	client   *goredis.Client
	password string
}

// Open creates a client. The server is contacted only when Ping is called.
func Open(cfg config.RedisConfig) *Client {
	return &Client{
		password: cfg.Password,
		client: goredis.NewClient(&goredis.Options{
			Addr:         cfg.Addr(),
			Password:     cfg.Password,
			DB:           cfg.DB,
			DialTimeout:  2 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		}),
	}
}

// Ping checks that Redis responds.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return database.Unavailable("redis", errRedisNotInitialized)
	}
	if err := c.client.Ping(ctx).Err(); err != nil {
		return database.Unavailable("redis", err, c.password)
	}
	return nil
}

// Close releases the client.
func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

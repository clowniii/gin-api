package redisrepo

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type Client struct{ *redis.Client }

var onceInstr sync.Once

func New(cfg Config) *Client {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB})
	// 尝试注册 otel tracing（幂等）
	onceInstr.Do(func() { _ = redisotel.InstrumentTracing(rdb) })
	return &Client{rdb}
}

func (c *Client) Ping(ctx context.Context) error { return c.Client.Ping(ctx).Err() }

func (c *Client) Close() error { return c.Client.Close() }

func (c *Client) SetTTL(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	return c.Client.Set(ctx, key, val, ttl).Err()
}

func (c *Client) Get(ctx context.Context, key string) string {
	res, err := c.Client.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return res
}
func (c *Client) Del(ctx context.Context, keys ...string) {
	_ = c.Client.Del(ctx, keys...).Err()
}

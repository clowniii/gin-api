package redisrepo

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PingTimeout  time.Duration
}

type Client struct{ *redis.Client }

var onceInstr sync.Once

func New(cfg Config) *Client {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB, DialTimeout: cfg.DialTimeout, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout})
	// 尝试注册 otel tracing（幂等）
	onceInstr.Do(func() { _ = redisotel.InstrumentTracing(rdb) })
	// 启动时 Ping (带超时) 便于尽早发现配置错误
	ctx, cancel := context.WithTimeout(context.Background(), cfg.PingTimeout)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		// 仅打印到标准输出（真正日志需在上层注入 logger，这里保持简洁）
		// 也可考虑返回包装 Client 以便上层记录
		// fmt.Printf("redis ping failed: %v\n", err)
	}
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

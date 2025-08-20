package cache

import (
	"context"
	"time"

	redisrepo "go-apiadmin/internal/repository/redis"
)

// RedisAdapter 实现 Cache 接口，包装 redis 客户端
// 假设 redis 存储的 value 已是 string（上层已 JSON 序列化）
// 追加 TTL 透传能力，用于 LayeredCache 回填 L1 时使用剩余 TTL

// TTLFetcher 可选接口：支持返回剩余 TTL
// 剩余 TTL <=0 代表永久或未知，不做回填 TTL 透传
// （不放入 Cache 基础接口，避免所有实现都修改）
type TTLFetcher interface {
	RemainingTTL(ctx context.Context, key string) (time.Duration, bool)
}

type RedisAdapter struct{ c *redisrepo.Client }

func NewRedisAdapter(c *redisrepo.Client) *RedisAdapter { return &RedisAdapter{c: c} }

func (r *RedisAdapter) Get(ctx context.Context, key string) (string, error) {
	return r.c.Get(ctx, key), nil
}
func (r *RedisAdapter) SetEX(ctx context.Context, key, val string, ttl time.Duration) error {
	return r.c.SetTTL(ctx, key, val, ttl)
}
func (r *RedisAdapter) Del(ctx context.Context, keys ...string) error {
	r.c.Del(ctx, keys...)
	return nil
}

// RemainingTTL 实现 TTLFetcher
func (r *RedisAdapter) RemainingTTL(ctx context.Context, key string) (time.Duration, bool) {
	// 利用 go-redis TTL 命令
	// 返回值: -2 key不存在; -1 无过期; 正常 >0
	res := r.c.Client.TTL(ctx, key)
	if err := res.Err(); err != nil {
		return 0, false
	}
	d := res.Val()
	if d <= 0 {
		return 0, false
	}
	return d, true
}

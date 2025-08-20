package cache

import (
	"context"
	"sync"
	"time"
)

// Cache 统一缓存接口（与 WikiCache 对齐并可替换 Redis 实现）
// value 统一以 string 形式存储（JSON 编解码在业务侧处理）
// 对于本地 L1 SimpleCache 我们内部会把 interface{} -> string 放入。
//
// SetEX: 设置带过期；Del: 删除多个 key。
// 若需要更丰富操作可再扩展。

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	SetEX(ctx context.Context, key, val string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// ===== SimpleCache (L1) 原实现 =====
// Item 泛型缓存条目
// 使用 interface{} 以避免引入 go1.18 泛型对现有工程的破坏
// 需要调用方自行做类型断言

type Item struct {
	Val interface{}
	Exp time.Time
}

// SimpleCache 一个线程安全、带 TTL 的简易进程级缓存。
// 注意：实现的是一个通用结构；要实现 Cache 接口需要封装。

// SimpleCache 本身用于通用场景；为实现 Cache 接口新增 simpleAdapter

type SimpleCache struct {
	mu   sync.RWMutex
	data map[string]Item
	ttl  time.Duration // 默认 TTL，可被 SetWithTTL 覆盖
}

func New(ttl time.Duration) *SimpleCache { return &SimpleCache{data: make(map[string]Item), ttl: ttl} }

// 原生对象方法（不直接满足 Cache 接口）
func (c *SimpleCache) getRaw(key string) (interface{}, bool) {
	c.mu.RLock()
	it, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !it.Exp.IsZero() && time.Now().After(it.Exp) {
		return nil, false
	}
	return it.Val, true
}
func (c *SimpleCache) setRaw(key string, val interface{}, ttl time.Duration) {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.data[key] = Item{Val: val, Exp: exp}
	c.mu.Unlock()
}
func (c *SimpleCache) delRaw(keys ...string) {
	c.mu.Lock()
	for _, k := range keys {
		delete(c.data, k)
	}
	c.mu.Unlock()
}
func (c *SimpleCache) Flush() { c.mu.Lock(); c.data = make(map[string]Item); c.mu.Unlock() }
func (c *SimpleCache) Keys() []string {
	c.mu.RLock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	c.mu.RUnlock()
	return keys
}

// 兼容之前使用方式：公开 Get/Set/Del（非接口版，返回 interface{}）
func (c *SimpleCache) Get(key string) (interface{}, bool) { return c.getRaw(key) }
func (c *SimpleCache) Set(key string, val interface{}) {
	c.setRaw(key, val, c.ttl)
}
func (c *SimpleCache) SetWithTTL(key string, val interface{}, ttl time.Duration) {
	c.setRaw(key, val, ttl)
}
func (c *SimpleCache) Del(keys ...string) {
	c.delRaw(keys...)
}

// ===== simpleAdapter: 将 SimpleCache 适配为 Cache 接口（键值都按 string） =====

type simpleAdapter struct{ c *SimpleCache }

func NewSimpleAdapter(c *SimpleCache) Cache { return &simpleAdapter{c: c} }

func (a *simpleAdapter) Get(_ context.Context, key string) (string, error) {
	if v, ok := a.c.getRaw(key); ok {
		if s, ok2 := v.(string); ok2 {
			return s, nil
		}
	}
	return "", nil
}
func (a *simpleAdapter) SetEX(_ context.Context, key, val string, ttl time.Duration) error {
	a.c.setRaw(key, val, ttl)
	return nil
}
func (a *simpleAdapter) Del(_ context.Context, keys ...string) error {
	a.c.delRaw(keys...)
	return nil
}

// RemainingTTL 提供与 RedisAdapter 一致的 TTL 查询能力，便于 LayeredCache 透传
// 若无过期时间或不存在返回 false
func (a *simpleAdapter) RemainingTTL(_ context.Context, key string) (time.Duration, bool) {
	a.c.mu.RLock()
	it, ok := a.c.data[key]
	a.c.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if it.Exp.IsZero() {
		return 0, false
	}
	if time.Now().After(it.Exp) {
		return 0, false
	}
	return time.Until(it.Exp), true
}

// ===== RedisAdapter: 适配已有 redis repo 客户端 =====
// 放在单独文件 redis_adapter.go 中实现。

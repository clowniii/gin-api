package cache

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go-apiadmin/internal/metrics"
)

// LayeredCache 组合 L1 (本地) + L2 (远程) 两层，遵循 Cache 接口
// 读：L1 -> L2 -> miss
// 写：直接写 L1 + L2
// Del：两层都删
// 指标：HitsL1 / HitsL2 / Miss / Set / Del / BackfillL1
// 提供快照方法 SnapshotMetrics 返回当前统计

type LayeredCache struct {
	L1 Cache
	L2 Cache

	// metrics 使用原子计数
	hitsL1     uint64
	hitsL2     uint64
	miss       uint64
	setOps     uint64
	delOps     uint64
	backfillL1 uint64
	reqTotal   uint64 // 新增: 总请求次数 (Get 调用计数)

	// singleflight map 防击穿
	mu    sync.Mutex
	locks map[string]*flight
}

// flight 表示单飞中的等待者
type flight struct {
	wait chan struct{}
}

type LayeredMetrics struct {
	HitsL1     uint64  `json:"hits_l1"`
	HitsL2     uint64  `json:"hits_l2"`
	Miss       uint64  `json:"miss"`
	SetOps     uint64  `json:"set_ops"`
	DelOps     uint64  `json:"del_ops"`
	BackfillL1 uint64  `json:"backfill_l1"`
	ReqTotal   uint64  `json:"req_total"` // 新增: 总 Get 次数
	HitRate    float64 `json:"hit_rate"`
}

func (m LayeredMetrics) computeHitRate() LayeredMetrics {
	total := m.HitsL1 + m.HitsL2 + m.Miss
	if total > 0 {
		m.HitRate = float64(m.HitsL1+m.HitsL2) / float64(total)
	}
	return m
}

func NewLayered(l1, l2 Cache) *LayeredCache {
	return &LayeredCache{L1: l1, L2: l2, locks: make(map[string]*flight)}
}

// doSingleflight 获取锁；返回是否需加载
func (c *LayeredCache) doSingleflight(key string) (load bool, release func()) {
	c.mu.Lock()
	if f, ok := c.locks[key]; ok {
		ch := f.wait
		c.mu.Unlock()
		<-ch // 等待首个加载者完成
		return false, func() {}
	}
	f := &flight{wait: make(chan struct{})}
	c.locks[key] = f
	c.mu.Unlock()
	return true, func() {
		c.mu.Lock()
		if ff, ok := c.locks[key]; ok {
			close(ff.wait)
			delete(c.locks, key)
		}
		c.mu.Unlock()
	}
}

// Sentinel 与空值穿透防护工具
const nilSentinel = "nil" // 统一占位值

// WrapNil 将空标识包装为 sentinel
func WrapNil(empty bool) string {
	if empty {
		return nilSentinel
	}
	return ""
}

// IsNilSentinel 判断缓存命中是否为空占位
func IsNilSentinel(v string) bool {
	return v == nilSentinel
}

// JitterTTL 对外导出，以便 service 级直接复用 (0~10% 抖动)
func JitterTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}
	j := rand.Int63n(int64(ttl) / 10)
	return ttl + time.Duration(j)
}

func (c *LayeredCache) Get(ctx context.Context, key string) (string, error) {
	atomic.AddUint64(&c.reqTotal, 1)
	if c.L1 != nil {
		if v, _ := c.L1.Get(ctx, key); v != "" {
			atomic.AddUint64(&c.hitsL1, 1)
			metrics.CacheLayerHits.WithLabelValues("l1").Inc()
			return v, nil
		}
	}
	if c.L2 != nil {
		if v, _ := c.L2.Get(ctx, key); v != "" {
			atomic.AddUint64(&c.hitsL2, 1)
			metrics.CacheLayerHits.WithLabelValues("l2").Inc()
			// 回填 L1，透传 TTL
			if c.L1 != nil {
				var ttl time.Duration = 30 * time.Second
				if tf, ok := c.L2.(interface {
					RemainingTTL(context.Context, string) (time.Duration, bool)
				}); ok {
					if d, ok2 := tf.RemainingTTL(ctx, key); ok2 && d > 0 {
						ttl = d
					}
				}
				_ = c.L1.SetEX(ctx, key, v, JitterTTL(ttl))
				atomic.AddUint64(&c.backfillL1, 1)
				metrics.CacheBackfill.Inc()
			}
			return v, nil
		}
	}
	// singleflight
	load, release := c.doSingleflight(key)
	if !load { // 等待已加载，重新读一次 L1
		if c.L1 != nil {
			if v, _ := c.L1.Get(ctx, key); v != "" {
				atomic.AddUint64(&c.hitsL1, 1)
				metrics.CacheLayerHits.WithLabelValues("l1").Inc()
				return v, nil
			}
		}
		atomic.AddUint64(&c.miss, 1)
		metrics.CacheMiss.Inc()
		return "", nil
	}
	defer release()
	atomic.AddUint64(&c.miss, 1)
	metrics.CacheMiss.Inc()
	return "", nil
}
func (c *LayeredCache) SetEX(ctx context.Context, key, val string, ttl time.Duration) error {
	if c.L1 != nil {
		_ = c.L1.SetEX(ctx, key, val, JitterTTL(ttl))
	}
	if c.L2 != nil {
		_ = c.L2.SetEX(ctx, key, val, JitterTTL(ttl))
	}
	atomic.AddUint64(&c.setOps, 1)
	metrics.CacheSet.Inc()
	return nil
}
func (c *LayeredCache) Del(ctx context.Context, keys ...string) error {
	if c.L1 != nil {
		_ = c.L1.Del(ctx, keys...)
	}
	if c.L2 != nil {
		_ = c.L2.Del(ctx, keys...)
	}
	atomic.AddUint64(&c.delOps, 1)
	metrics.CacheDel.Inc()
	return nil
}

func (c *LayeredCache) SnapshotMetrics() LayeredMetrics {
	m := LayeredMetrics{
		HitsL1:     atomic.LoadUint64(&c.hitsL1),
		HitsL2:     atomic.LoadUint64(&c.hitsL2),
		Miss:       atomic.LoadUint64(&c.miss),
		SetOps:     atomic.LoadUint64(&c.setOps),
		DelOps:     atomic.LoadUint64(&c.delOps),
		BackfillL1: atomic.LoadUint64(&c.backfillL1),
		ReqTotal:   atomic.LoadUint64(&c.reqTotal), // 新增
	}
	return m.computeHitRate()
}

// ResetMetrics 重置当前指标计数（用于测试或阶段性观测）
func (c *LayeredCache) ResetMetrics() {
	atomic.StoreUint64(&c.hitsL1, 0)
	atomic.StoreUint64(&c.hitsL2, 0)
	atomic.StoreUint64(&c.miss, 0)
	atomic.StoreUint64(&c.setOps, 0)
	atomic.StoreUint64(&c.delOps, 0)
	atomic.StoreUint64(&c.backfillL1, 0)
	atomic.StoreUint64(&c.reqTotal, 0) // 新增
}

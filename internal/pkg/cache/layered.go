package cache

import (
	"context"
	"sync/atomic"
	"time"
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

func NewLayered(l1, l2 Cache) *LayeredCache { return &LayeredCache{L1: l1, L2: l2} }

func (c *LayeredCache) Get(ctx context.Context, key string) (string, error) {
	atomic.AddUint64(&c.reqTotal, 1) // 新增: 总请求计数
	if c.L1 != nil {
		if v, _ := c.L1.Get(ctx, key); v != "" {
			atomic.AddUint64(&c.hitsL1, 1)
			return v, nil
		}
	}
	if c.L2 != nil {
		if v, _ := c.L2.Get(ctx, key); v != "" {
			atomic.AddUint64(&c.hitsL2, 1)
			// 回填 L1，尝试透传剩余 TTL
			if c.L1 != nil {
				var ttl time.Duration = 30 * time.Second // 默认兜底
				if tf, ok := c.L2.(interface {
					RemainingTTL(context.Context, string) (time.Duration, bool)
				}); ok {
					if d, ok2 := tf.RemainingTTL(ctx, key); ok2 && d > 0 {
						ttl = d
					}
				}
				_ = c.L1.SetEX(ctx, key, v, ttl)
				atomic.AddUint64(&c.backfillL1, 1)
			}
			return v, nil
		}
	}
	atomic.AddUint64(&c.miss, 1)
	return "", nil
}
func (c *LayeredCache) SetEX(ctx context.Context, key, val string, ttl time.Duration) error {
	if c.L1 != nil {
		_ = c.L1.SetEX(ctx, key, val, ttl)
	}
	if c.L2 != nil {
		_ = c.L2.SetEX(ctx, key, val, ttl)
	}
	atomic.AddUint64(&c.setOps, 1)
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

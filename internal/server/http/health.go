package http

import (
	"context"
	"sync"
	"time"

	"go-apiadmin/internal/discovery/etcd"
	"go-apiadmin/internal/metrics"
	"go-apiadmin/internal/mq/kafka"
	redisrepo "go-apiadmin/internal/repository/redis"

	"gorm.io/gorm"
)

// HealthChecker 聚合健康检查（liveness / readiness）
type HealthChecker struct {
	db       *gorm.DB
	redis    *redisrepo.Client
	producer *kafka.Producer
	etcdCli  *etcd.Client

	cacheMu     sync.Mutex
	cacheResult map[string]interface{}
	cacheExpiry time.Time
	cacheTTL    time.Duration
}

func NewHealthChecker(db *gorm.DB, r *redisrepo.Client, p *kafka.Producer, e *etcd.Client) *HealthChecker {
	return &HealthChecker{db: db, redis: r, producer: p, etcdCli: e, cacheTTL: 2 * time.Second}
}

// Liveness 仅表示进程活着，不依赖外部组件
func (h *HealthChecker) Liveness() map[string]interface{} {
	return map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}
}

// Readiness 检测外部依赖，带缓存与耗时指标
func (h *HealthChecker) Readiness(ctx context.Context) (map[string]interface{}, int) {
	// 缓存命中
	h.cacheMu.Lock()
	if time.Now().Before(h.cacheExpiry) && h.cacheResult != nil {
		res := h.cacheResult
		h.cacheMu.Unlock()
		statusCode := 200
		if res["status"] != "ok" { // degraded
			statusCode = 503
		}
		return res, statusCode
	}
	h.cacheMu.Unlock()

	res := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
		"detail": []map[string]interface{}{},
	}

	type depResult struct {
		name string
		up   bool
		err  string
		dur  time.Duration
	}
	deps := []string{"db", "redis", "kafka", "etcd"}
	results := make(chan depResult, len(deps))
	var wg sync.WaitGroup
	wg.Add(len(deps))

	checkWithTimeout := func(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
		ctx2, cancel := context.WithTimeout(parent, timeout)
		return ctx2, cancel
	}

	// DB
	go func() {
		defer wg.Done()
		start := time.Now()
		out := depResult{name: "db"}
		if h.db != nil {
			if sqlDB, err := h.db.DB(); err == nil {
				ctx2, cancel := checkWithTimeout(ctx, 300*time.Millisecond)
				if err := sqlDB.PingContext(ctx2); err == nil {
					out.up = true
				} else {
					out.err = err.Error()
				}
				cancel()
			} else {
				out.err = err.Error()
			}
		} else {
			out.err = "nil"
		}
		out.dur = time.Since(start)
		metrics.DependencyCheckDuration.WithLabelValues("db").Observe(out.dur.Seconds())
		if out.up {
			metrics.DBUp.Set(1)
		} else {
			metrics.DBUp.Set(0)
		}
		results <- out
	}()
	// Redis
	go func() {
		defer wg.Done()
		start := time.Now()
		out := depResult{name: "redis"}
		if h.redis != nil {
			ctx2, cancel := checkWithTimeout(ctx, 250*time.Millisecond)
			if err := h.redis.Ping(ctx2); err == nil {
				out.up = true
			} else {
				out.err = err.Error()
			}
			cancel()
		} else {
			out.err = "nil"
		}
		out.dur = time.Since(start)
		metrics.DependencyCheckDuration.WithLabelValues("redis").Observe(out.dur.Seconds())
		if out.up {
			metrics.RedisUp.Set(1)
		} else {
			metrics.RedisUp.Set(0)
		}
		results <- out
	}()
	// Kafka
	go func() {
		defer wg.Done()
		start := time.Now()
		out := depResult{name: "kafka"}
		if h.producer != nil {
			ctx2, cancel := checkWithTimeout(ctx, 250*time.Millisecond)
			if err := h.producer.WriteMessages(ctx2); err == nil {
				out.up = true
			} else {
				out.err = err.Error()
			}
			cancel()
		} else {
			out.err = "nil"
		}
		out.dur = time.Since(start)
		metrics.DependencyCheckDuration.WithLabelValues("kafka").Observe(out.dur.Seconds())
		if out.up {
			metrics.KafkaUp.Set(1)
		} else {
			metrics.KafkaUp.Set(0)
		}
		results <- out
	}()
	// Etcd
	go func() {
		defer wg.Done()
		start := time.Now()
		out := depResult{name: "etcd"}
		if h.etcdCli != nil {
			ctx2, cancel := checkWithTimeout(ctx, 250*time.Millisecond)
			if _, err := h.etcdCli.Get(ctx2, "health"); err == nil {
				out.up = true
			} else {
				out.err = err.Error()
			}
			cancel()
		} else {
			out.err = "nil"
		}
		out.dur = time.Since(start)
		metrics.DependencyCheckDuration.WithLabelValues("etcd").Observe(out.dur.Seconds())
		if out.up {
			metrics.EtcdUp.Set(1)
		} else {
			metrics.EtcdUp.Set(0)
		}
		results <- out
	}()

	wg.Wait()
	close(results)

	upTotal := 0
	for r := range results {
		if r.up {
			res[r.name] = "up"
			upTotal++
		} else {
			if r.err == "" {
				res[r.name] = "down"
			} else {
				res[r.name] = r.err
			}
		}
		res[r.name+"_duration_ms"] = float64(r.dur.Microseconds()) / 1000.0
		res["detail"] = append(res["detail"].([]map[string]interface{}), map[string]interface{}{"dep": r.name, "up": r.up, "error": r.err, "duration_ms": float64(r.dur.Microseconds()) / 1000.0})
	}
	if upTotal < len(deps) {
		res["status"] = "degraded"
	}

	// 写缓存
	h.cacheMu.Lock()
	h.cacheResult = res
	h.cacheExpiry = time.Now().Add(h.cacheTTL)
	h.cacheMu.Unlock()

	statusCode := 200
	if res["status"] != "ok" {
		statusCode = 503
	}
	return res, statusCode
}

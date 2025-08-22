package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency distribution",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "method"})
	RequestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"path", "method", "status"})
	Inflight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_inflight_requests",
		Help: "In-flight HTTP requests",
	})
	DBUp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_up",
		Help: "Database connectivity (1=up,0=down)",
	})
	RedisUp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "redis_up",
		Help: "Redis connectivity (1=up,0=down)",
	})
	KafkaUp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_up",
		Help: "Kafka connectivity (1=up,0=down)",
	})
	EtcdUp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "etcd_up",
		Help: "Etcd connectivity (1=up,0=down)",
	})
	DependencyCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dependency_check_duration_seconds",
		Help:    "Latency of dependency health checks",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.4, 0.8, 1},
	}, []string{"dep"})
	AuthActionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_action_total",
		Help: "Total auth actions (login/refresh)",
	}, []string{"action", "result"})
	AuthActionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auth_action_duration_seconds",
		Help:    "Latency of auth actions",
		Buckets: prometheus.DefBuckets,
	}, []string{"action", "result"})
	CacheLayerHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_layer_hits_total",
		Help: "Layered cache hits by layer (l1/l2)",
	}, []string{"layer"})
	CacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_miss_total",
		Help: "Layered cache miss count",
	})
	CacheSet = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_set_total",
		Help: "Cache set operations",
	})
	CacheDel = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_del_total",
		Help: "Cache delete operations",
	})
	CacheBackfill = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_backfill_l1_total",
		Help: "Backfill L1 from L2 count",
	})
	CacheNilHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_nil_sentinel_hit_total",
		Help: "Hits of nil sentinel (empty protection)",
	})
	PermissionInvalidateTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "permission_invalidate_total",
		Help: "Permission cache invalidation count by mode (single/group/all)",
	}, []string{"mode"})
	PermissionInvalidateUsersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "permission_invalidate_users_total",
		Help: "Total users affected by permission group invalidations",
	})
	HTTPAccessKafkaEnqueue = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_access_kafka_enqueue_total",
		Help: "Enqueued http access log messages to internal async queue",
	}, []string{"result"}) // result=ok|dropped
	HTTPAccessKafkaQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_access_kafka_queue_depth",
		Help: "Current depth of http access async queue",
	})
	HTTPAccessKafkaSendDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_access_kafka_send_duration_seconds",
		Help:    "Duration of kafka send for http access logs (single or batch loop time)",
		Buckets: []float64{0.001, 0.003, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2},
	})
	HTTPAccessKafkaErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "http_access_kafka_errors_total",
		Help: "Errors during kafka send for http access logs",
	})
	HTTPAccessKafkaBatchFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_access_kafka_batch_flush_total",
		Help: "Batch flush count by reason (size|timeout|shutdown)",
	}, []string{"reason"})
	HTTPAccessKafkaBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_access_kafka_batch_size",
		Help:    "Number of messages per batch flush",
		Buckets: []float64{1, 2, 5, 10, 20, 50, 100, 200},
	})
	HTTPAccessKafkaFlushDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_access_kafka_flush_duration_seconds",
		Help:    "Duration of batch flush by reason (size|timeout|shutdown)",
		Buckets: []float64{0.0005, 0.001, 0.002, 0.003, 0.005, 0.01, 0.02, 0.05, 0.1},
	}, []string{"reason"})
	HTTPAccessKafkaQueueDelayAvg = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_access_kafka_queue_delay_avg_seconds",
		Help:    "Average queue delay per batch (enqueue to flush)",
		Buckets: []float64{0.0005, 0.001, 0.002, 0.003, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2},
	})
	HTTPAccessKafkaQueueDelayMax = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_access_kafka_queue_delay_max_seconds",
		Help:    "Max message queue delay per batch (enqueue to flush)",
		Buckets: []float64{0.0005, 0.001, 0.002, 0.003, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2},
	})
	// ===== 新增认证细化指标 =====
	AuthSessionCacheHit = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_session_cache_hit_total",
		Help: "Auth session cache hits by action (login/refresh/userinfo)",
	}, []string{"action"})
	AuthSessionCacheSet = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_session_cache_set_total",
		Help: "Auth session cache set operations by action",
	}, []string{"action"})
	AuthRefreshRotateTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_refresh_rotate_total",
		Help: "Total rotated refresh tokens",
	})
)

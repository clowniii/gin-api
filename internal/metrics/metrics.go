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
)

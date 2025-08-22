package observability

import (
	"encoding/json"
	"time"

	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/mq/kafka"

	"github.com/gin-gonic/gin"
)

// AccessLogKafka 将基础 HTTP 访问信息发送到 Kafka (同步发送)。
func AccessLogKafka(l *logging.Logger, p *kafka.Producer) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if p == nil { // 保护
			return
		}
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		ua := c.Request.UserAgent()
		if len(ua) > 200 {
			ua = ua[:200]
		}
		entry := map[string]interface{}{
			"type":       "http_access",
			"path":       path,
			"method":     c.Request.Method,
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
			"ip":         c.ClientIP(),
			"ts":         time.Now().Unix(),
			"ua":         ua,
		}
		if v, ok := c.Get("trace_id"); ok {
			entry["trace_id"] = v
		}
		if uid, ok := c.Get("user_id"); ok {
			entry["user_id"] = uid
		}
		b, _ := json.Marshal(entry)
		if traceID, ok := entry["trace_id"].(string); ok && traceID != "" {
			_ = p.SendWithHeaders(c.Request.Context(), nil, b, map[string]string{"trace_id": traceID})
		} else {
			_ = p.Send(c.Request.Context(), nil, b)
		}
	}
}

// AccessLogKafkaAsync 异步批量发送版本，根据 access_kafka_async 配置启用。
func AccessLogKafkaAsync(l *logging.Logger, sender *kafka.AccessAsyncSender) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if sender == nil {
			return
		}
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		ua := c.Request.UserAgent()
		if len(ua) > 200 {
			ua = ua[:200]
		}
		entry := map[string]interface{}{
			"type":       "http_access",
			"path":       path,
			"method":     c.Request.Method,
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
			"ip":         c.ClientIP(),
			"ts":         time.Now().Unix(),
			"ua":         ua,
		}
		if v, ok := c.Get("trace_id"); ok {
			entry["trace_id"] = v
		}
		if uid, ok := c.Get("user_id"); ok {
			entry["user_id"] = uid
		}
		b, _ := json.Marshal(entry)
		var headers map[string]string
		if traceID, ok := entry["trace_id"].(string); ok && traceID != "" {
			headers = map[string]string{"trace_id": traceID}
		}
		sender.Enqueue(kafka.AsyncMessage{Ctx: c.Request.Context(), Value: b, Headers: headers, EnqueueAt: time.Now()})
	}
}

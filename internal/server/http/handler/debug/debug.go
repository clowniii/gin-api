package debug

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go-apiadmin/internal/config"
	"go-apiadmin/internal/logging"

	"github.com/gin-gonic/gin"
	kafkaGo "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Dependencies 仅用于调试 handler
type Dependencies struct {
	Config *config.Config
	Logger *logging.Logger
}

type Handler struct{ d Dependencies }

func New(d Dependencies) *Handler { return &Handler{d: d} }

// PeekAccessLog 从 Kafka op_log_topic 读取一条消息 (http_access 或 operation log 均可) 用于链路调试。
// 每次请求都会创建临时 reader，避免常驻 goroutine。读取超时返回 code -2。
func (h *Handler) PeekAccessLog(c *gin.Context) {
	Second := c.Param("Second")
	//string 换 int
	second, _ := strconv.Atoi(Second)
	cfg := h.d.Config
	if cfg == nil || len(cfg.Kafka.Brokers) == 0 || cfg.Kafka.OpLogTopic == "" {
		c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "kafka 未配置", "data": gin.H{}})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(second)*time.Millisecond)
	defer cancel()
	reader := kafkaGo.NewReader(kafkaGo.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.OpLogTopic,
		GroupID:  "debug-access-peek", // 共享 group 便于轮询
		MinBytes: 1 << 10,
		MaxBytes: 1 << 20,
		MaxWait:  200 * time.Millisecond,
	})
	defer reader.Close()

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -2, "msg": "读取超时或错误", "data": gin.H{"error": err.Error()}})
		return
	}
	// headers -> map
	headers := make(map[string]string, len(msg.Headers))
	for _, hkv := range msg.Headers {
		headers[hkv.Key] = string(hkv.Value)
	}
	// trace 提取
	carrier := propagation.MapCarrier{}
	for k, v := range headers {
		carrier[k] = v
	}
	extractedCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
	sc := trace.SpanContextFromContext(extractedCtx)
	traceID := ""
	spanID := ""
	if sc.IsValid() {
		traceID = sc.TraceID().String()
		spanID = sc.SpanID().String()
	}
	var body map[string]interface{}
	_ = json.Unmarshal(msg.Value, &body)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"kafka_topic": cfg.Kafka.OpLogTopic,
			"trace_id":    traceID,
			"span_id":     spanID,
			"headers":     headers,
			"raw_body":    string(msg.Value),
			"body":        body,
			"partition":   msg.Partition,
			"offset":      msg.Offset,
			"key":         string(msg.Key),
			"ingest_unix": time.Now().Unix(),
		},
	})
}

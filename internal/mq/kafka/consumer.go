package kafka

import (
	"context"
	"errors"
	"log"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type ConsumerConfig struct {
	Brokers        []string
	GroupID        string
	Topics         []string
	MinBytes       int
	MaxBytes       int
	CommitInterval time.Duration
}

type MessageHandler func(ctx context.Context, msg kafkaGo.Message) error

type Consumer struct {
	reader *kafkaGo.Reader
}

// context key 定义，避免使用裸 string
type ctxKey string

const traceIDKey ctxKey = "trace_id"

func NewConsumer(cfg ConsumerConfig) *Consumer {
	if cfg.MinBytes == 0 {
		cfg.MinBytes = 1 << 10
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10 << 20
	}
	if cfg.CommitInterval == 0 {
		cfg.CommitInterval = time.Second
	}
	reader := kafkaGo.NewReader(kafkaGo.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		GroupTopics:    cfg.Topics,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		CommitInterval: cfg.CommitInterval,
	})
	return &Consumer{reader: reader}
}

// Start 消费循环：
// 1. 提取 W3C traceparent / baggage 形成上下文
// 2. 创建 kafka.consume Span (SpanKindConsumer)
// 3. 兼容旧 trace_id header -> ctx value
// 4. 调用业务 handler，记录错误到 Span
func (c *Consumer) Start(ctx context.Context, handler MessageHandler) error {
	if c.reader == nil {
		return errors.New("nil reader")
	}
	prop := otel.GetTextMapPropagator()
	tracer := otel.Tracer("kafka-consumer")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		// --- 构造 carrier 并提取上下文 ---
		carrier := propagation.MapCarrier{}
		for _, h := range m.Headers {
			carrier[h.Key] = string(h.Value)
		}
		msgCtx := prop.Extract(ctx, carrier)
		// 兼容旧 trace_id（若存在）
		if v, ok := carrier["trace_id"]; ok && v != "" {
			msgCtx = context.WithValue(msgCtx, traceIDKey, v)
		}

		// 创建消费 Span
		attrs := []attribute.KeyValue{
			semconv.MessagingSystem("kafka"),
			semconv.MessagingDestinationName(m.Topic),
			attribute.String("messaging.destination_kind", "topic"),
			attribute.Int("messaging.kafka.partition", m.Partition),
			attribute.Int64("messaging.kafka.offset", m.Offset),
			attribute.Int("messaging.message.key_size", len(m.Key)),
			attribute.Int("messaging.message.size", len(m.Value)),
		}
		msgCtx, span := tracer.Start(msgCtx, "kafka.consume", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attrs...))

		if err := handler(msgCtx, m); err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			log.Printf("kafka consumer handler error: %v", err)
		}
		span.End()
	}
}

func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

package kafka

import (
	"context"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Brokers []string
	Topic   string
}

// Producer 封装 kafka-go Writer，新增 OpenTelemetry 发送埋点
type Producer struct{ *kafkaGo.Writer }

func NewProducer(cfg Config) *Producer {
	w := &kafkaGo.Writer{
		Addr:         kafkaGo.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		RequiredAcks: kafkaGo.RequireOne,
		BatchTimeout: 10 * time.Millisecond,
	}
	return &Producer{w}
}

// StartSpan 创建发送 span（若存在父 span 则关联）
func (p *Producer) startSpan(ctx context.Context) (context.Context, trace.Span) {
	tr := otel.GetTracerProvider().Tracer("kafka-producer")
	attrs := []attribute.KeyValue{
		semconv.MessagingSystem("kafka"),
		semconv.MessagingDestinationName(p.Topic),
		attribute.String("messaging.destination_kind", "topic"),
	}
	return tr.Start(ctx, "kafka.produce", trace.WithSpanKind(trace.SpanKindProducer), trace.WithAttributes(attrs...))
}

func (p *Producer) injectHeaders(ctx context.Context, headers []kafkaGo.Header) []kafkaGo.Header {
	// 使用 W3C propagator 注入 traceparent / baggage
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	// 防重复: 若已存在同 key 则跳过注入
	existing := make(map[string]struct{}, len(headers))
	for _, h := range headers {
		existing[h.Key] = struct{}{}
	}
	for k, v := range carrier {
		if _, ok := existing[k]; ok {
			continue
		}
		headers = append(headers, kafkaGo.Header{Key: k, Value: []byte(v)})
	}
	return headers
}

func (p *Producer) Send(ctx context.Context, key, value []byte) error {
	ctx, span := p.startSpan(ctx)
	defer span.End()
	msg := kafkaGo.Message{Key: key, Value: value, Time: time.Now()}
	msg.Headers = p.injectHeaders(ctx, msg.Headers)
	if err := p.Writer.WriteMessages(ctx, msg); err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}
	return nil
}

// SendWithHeaders 支持自定义 headers (附带 trace、user_id 等) 并自动注入 trace 上下文
func (p *Producer) SendWithHeaders(ctx context.Context, key, value []byte, headers map[string]string) error {
	ctx, span := p.startSpan(ctx)
	defer span.End()
	hs := make([]kafkaGo.Header, 0, len(headers))
	for k, v := range headers {
		hs = append(hs, kafkaGo.Header{Key: k, Value: []byte(v)})
	}
	hs = p.injectHeaders(ctx, hs)
	msg := kafkaGo.Message{Key: key, Value: value, Time: time.Now(), Headers: hs}
	if err := p.Writer.WriteMessages(ctx, msg); err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}
	return nil
}

func (p *Producer) Close() error { return p.Writer.Close() }

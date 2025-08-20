package kafka

import (
	"context"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
)

type Config struct {
	Brokers []string
	Topic   string
}

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

func (p *Producer) Send(ctx context.Context, key, value []byte) error {
	return p.Writer.WriteMessages(ctx, kafkaGo.Message{Key: key, Value: value})
}

// SendWithHeaders 支持自定义 headers (附带 trace、user_id 等)
func (p *Producer) SendWithHeaders(ctx context.Context, key, value []byte, headers map[string]string) error {
	hs := make([]kafkaGo.Header, 0, len(headers))
	for k, v := range headers {
		hs = append(hs, kafkaGo.Header{Key: k, Value: []byte(v)})
	}
	return p.Writer.WriteMessages(ctx, kafkaGo.Message{Key: key, Value: value, Time: time.Now(), Headers: hs})
}

func (p *Producer) Close() error { return p.Writer.Close() }

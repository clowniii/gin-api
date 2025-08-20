package kafka

import (
	"context"
	"errors"
	"log"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
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
	reader  *kafkaGo.Reader
	closers []func() error
}

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

// Start 消费循环，自动解析 trace_id header 注入 context
func (c *Consumer) Start(ctx context.Context, handler MessageHandler) error {
	if c.reader == nil {
		return errors.New("nil reader")
	}
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		msgCtx := ctx
		// 提取 trace_id
		for _, h := range m.Headers {
			if h.Key == "trace_id" {
				msgCtx = context.WithValue(msgCtx, "trace_id", string(h.Value))
				break
			}
		}
		if err := handler(msgCtx, m); err != nil {
			log.Printf("kafka consumer handler error: %v", err)
		}
	}
}

func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

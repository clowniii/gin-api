package oplog

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/repository/postgres"

	kafkaGo "github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

type Consumer struct {
	cfg    Config
	reader *kafkaGo.Reader
	DB     *gorm.DB
}

type OpLogEntry struct {
	Path      string   `json:"path"`
	Method    string   `json:"method"`
	Status    int      `json:"status"`
	LatencyMs int64    `json:"latency_ms"`
	IP        string   `json:"ip"`
	UserID    int64    `json:"user_id"`
	Time      string   `json:"time"`
	Body      string   `json:"body"`
	Errors    []string `json:"errors,omitempty"`
}

func NewConsumer(cfg Config, db *gorm.DB) *Consumer {
	reader := kafkaGo.NewReader(kafkaGo.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 1, MaxBytes: 10e6,
	})
	return &Consumer{cfg: cfg, reader: reader, DB: db}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var e OpLogEntry
		if err := json.Unmarshal(m.Value, &e); err != nil {
			log.Printf("oplog consumer unmarshal err: %v", err)
			continue
		}
		// 转换时间
		var ts int64
		if t, err := time.Parse(time.RFC3339, e.Time); err == nil {
			ts = t.Unix()
		} else {
			ts = time.Now().Unix()
		}
		rec := model.AdminUserAction{
			ActionName: "", // 可后续由路径映射
			UID:        e.UserID,
			Nickname:   "", // 可后续补充
			AddTime:    ts,
			Data:       truncate(e.Body, 2000),
			URL:        e.Path,
			Method:     e.Method,
			Status:     e.Status,
			LatencyMs:  e.LatencyMs,
			IP:         e.IP,
		}
		if err := c.DB.WithContext(ctx).Create(&rec).Error; err != nil {
			log.Printf("oplog consumer save err: %v", err)
		}
	}
}

func (c *Consumer) Close() error { return c.reader.Close() }

// 若需要迁移表结构，可在初始化时调用
func AutoMigrate(db *gorm.DB) error {
	return postgres.AutoMigrateModels(db, &model.AdminUserAction{})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	HTTP struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"http"`
	Postgres struct {
		DSN         string `mapstructure:"dsn"`
		MaxOpen     int    `mapstructure:"max_open"`
		MaxIdle     int    `mapstructure:"max_idle"`
		AutoMigrate bool   `mapstructure:"auto_migrate"`
	} `mapstructure:"postgres"`
	Redis struct {
		Addr           string `mapstructure:"addr"`
		Password       string `mapstructure:"password"`
		DB             int    `mapstructure:"db"`
		JTIPrefix      string `mapstructure:"jti_prefix"`
		DialTimeoutMS  int    `mapstructure:"dial_timeout_ms"`
		ReadTimeoutMS  int    `mapstructure:"read_timeout_ms"`
		WriteTimeoutMS int    `mapstructure:"write_timeout_ms"`
		PingTimeoutMS  int    `mapstructure:"ping_timeout_ms"`
	} `mapstructure:"redis"`
	Kafka struct {
		Brokers    []string `mapstructure:"brokers"`
		OpLogTopic string   `mapstructure:"op_log_topic"`
	} `mapstructure:"kafka"`
	Etcd struct {
		Endpoints []string `mapstructure:"endpoints"`
		TTL       int      `mapstructure:"ttl"`
	} `mapstructure:"etcd"`
	JWT struct {
		Secret        string `mapstructure:"secret"`
		ExpireSeconds int    `mapstructure:"expire_seconds"`
		Issuer        string `mapstructure:"issuer"`
	} `mapstructure:"jwt"`
	Auth struct { // 新增: 认证/会话相关配置
		SessionTTLSeconds int  `mapstructure:"session_ttl_seconds"`
		RotateRefresh     bool `mapstructure:"rotate_refresh"` // 是否在 refresh 时旋转 refresh token
	} `mapstructure:"auth"`
	Log struct {
		Level            string `mapstructure:"level"`
		Format           string `mapstructure:"format"`
		AccessKafka      bool   `mapstructure:"access_kafka"`
		AccessKafkaAsync struct {
			Enable    bool `mapstructure:"enable"`
			QueueSize int  `mapstructure:"queue_size"`
			Workers   int  `mapstructure:"workers"`
			Batch     struct {
				MaxMsgs   int `mapstructure:"max_msgs"`
				MaxWaitMS int `mapstructure:"max_wait_ms"`
			} `mapstructure:"batch"`
		} `mapstructure:"access_kafka_async"`
	} `mapstructure:"log"`
	AppMeta struct {
		Name    string `mapstructure:"name"`
		Version string `mapstructure:"version"`
		Env     string `mapstructure:"env"` // 新增: 运行环境 dev|staging|prod 等
	} `mapstructure:"app_meta"`
	Wiki struct {
		OnlineTimeSeconds int `mapstructure:"online_time_seconds"`
	} `mapstructure:"wiki"`
	Upload struct { // 新增: 上传相关限制
		MaxSizeMB  int      `mapstructure:"max_size_mb"`
		AllowedExt []string `mapstructure:"allowed_ext"`
	} `mapstructure:"upload"`
	OTel struct {
		Endpoint     string  `mapstructure:"endpoint"` // OTLP gRPC/HTTP endpoint
		Insecure     bool    `mapstructure:"insecure"`
		SamplerRatio float64 `mapstructure:"sampler_ratio"`
		Enable       bool    `mapstructure:"enable"`
	} `mapstructure:"otel"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	// 默认值
	v.SetDefault("wiki.online_time_seconds", 86400)
	v.SetDefault("app_meta.name", "GOAPIAdmin")
	v.SetDefault("app_meta.version", "v1")
	v.SetDefault("app_meta.env", "dev")
	v.SetDefault("upload.max_size_mb", 10)
	v.SetDefault("upload.allowed_ext", []string{"jpg", "jpeg", "png", "gif", "pdf", "txt", "zip", "json"})
	v.SetDefault("otel.enable", false)
	v.SetDefault("otel.sampler_ratio", 1.0)
	v.SetDefault("otel.insecure", true)
	v.SetDefault("log.access_kafka", false)
	v.SetDefault("log.access_kafka_async.enable", false)
	v.SetDefault("log.access_kafka_async.queue_size", 10000)
	v.SetDefault("log.access_kafka_async.workers", 2)
	v.SetDefault("log.access_kafka_async.batch.max_msgs", 50)
	v.SetDefault("log.access_kafka_async.batch.max_wait_ms", 20)
	// Redis 时间相关默认（毫秒）
	v.SetDefault("redis.dial_timeout_ms", 800)
	v.SetDefault("redis.read_timeout_ms", 500)
	v.SetDefault("redis.write_timeout_ms", 500)
	v.SetDefault("redis.ping_timeout_ms", 500)
	// Auth 默认
	v.SetDefault("auth.session_ttl_seconds", 300) // 5 分钟
	v.SetDefault("auth.rotate_refresh", true)
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	// ===== 逻辑校验 =====
	if c.HTTP.Addr == "" {
		return nil, errors.New("http.addr required")
	}
	if c.JWT.Secret == "" || len(c.JWT.Secret) < 16 {
		return nil, fmt.Errorf("jwt.secret too short (>=16)")
	}
	if c.JWT.ExpireSeconds <= 0 {
		return nil, fmt.Errorf("jwt.expire_seconds must >0")
	}
	if c.OTel.Enable {
		if c.OTel.Endpoint == "" {
			return nil, errors.New("otel.endpoint required when otel.enable=true")
		}
		if c.OTel.SamplerRatio < 0 || c.OTel.SamplerRatio > 1 {
			return nil, errors.New("otel.sampler_ratio must be in [0,1]")
		}
	}
	if len(c.Redis.JTIPrefix) == 0 {
		c.Redis.JTIPrefix = "jwt:jti:"
	}
	if c.AppMeta.Env == "" {
		c.AppMeta.Env = "dev"
	}
	// AccessKafkaAsync 合法化
	if c.Log.AccessKafkaAsync.Batch.MaxMsgs <= 0 {
		c.Log.AccessKafkaAsync.Batch.MaxMsgs = 50
	}
	if c.Log.AccessKafkaAsync.Batch.MaxWaitMS <= 0 {
		c.Log.AccessKafkaAsync.Batch.MaxWaitMS = 20
	}
	if c.Log.AccessKafkaAsync.QueueSize <= 0 {
		c.Log.AccessKafkaAsync.QueueSize = 10000
	}
	if c.Log.AccessKafkaAsync.Workers <= 0 {
		c.Log.AccessKafkaAsync.Workers = 2
	}
	// Redis 超时容错（防止 0/负数）
	if c.Redis.DialTimeoutMS <= 0 {
		c.Redis.DialTimeoutMS = 800
	}
	if c.Redis.ReadTimeoutMS <= 0 {
		c.Redis.ReadTimeoutMS = 500
	}
	if c.Redis.WriteTimeoutMS <= 0 {
		c.Redis.WriteTimeoutMS = 500
	}
	if c.Redis.PingTimeoutMS <= 0 {
		c.Redis.PingTimeoutMS = 500
	}
	if c.Auth.SessionTTLSeconds <= 0 { // 容错
		c.Auth.SessionTTLSeconds = 300
	}
	return &c, nil
}

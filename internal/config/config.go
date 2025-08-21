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
		Addr      string `mapstructure:"addr"`
		Password  string `mapstructure:"password"`
		DB        int    `mapstructure:"db"`
		JTIPrefix string `mapstructure:"jti_prefix"`
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
	Log struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"log"`
	AppMeta struct {
		Name    string `mapstructure:"name"`
		Version string `mapstructure:"version"`
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
	v.SetDefault("upload.max_size_mb", 10)
	v.SetDefault("upload.allowed_ext", []string{"jpg", "jpeg", "png", "gif", "pdf", "txt", "zip", "json"})
	v.SetDefault("otel.enable", false)
	v.SetDefault("otel.sampler_ratio", 1.0)
	v.SetDefault("otel.insecure", true)
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
	return &c, nil
}

package logging

import (
	"context"

	"go.uber.org/zap"
)

type Logger struct {
	*zap.Logger
}

func New(level, format string) (*Logger, error) {
	var cfg zap.Config
	if format == "console" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	if level != "" {
		if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
			return nil, err
		}
	}
	lg, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return &Logger{lg}, nil
}

// 简化 main 中调用
func (l *Logger) Info(msg string, fields ...interface{}) {
	fs := make([]zap.Field, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		if f, ok := fields[i].(interface {
			Key() string
			Value() interface{}
		}); ok {
			fs = append(fs, zap.Any(f.Key(), f.Value()))
		}
	}
	l.Logger.Info(msg, fs...)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	fs := make([]zap.Field, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		if f, ok := fields[i].(interface {
			Key() string
			Value() interface{}
		}); ok {
			fs = append(fs, zap.Any(f.Key(), f.Value()))
		}
	}
	l.Logger.Error(msg, fs...)
}

func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return l.Logger
	}
	fields := make([]zap.Field, 0, 2)
	if v := ctx.Value("trace_id"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			fields = append(fields, zap.String("trace_id", s))
		}
	}
	if v := ctx.Value("user_id"); v != nil {
		if id, ok := v.(int64); ok && id > 0 {
			fields = append(fields, zap.Int64("user_id", id))
		}
	}
	if len(fields) == 0 {
		return l.Logger
	}
	return l.Logger.With(fields...)
}

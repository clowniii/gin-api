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

// Info 直接接受 zap.Field，避免 interface{} 误传
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Error 直接接受 zap.Field
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// WithContext 依据上下文注入 trace_id / user_id 字段，返回新的 *Logger（链式）
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
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
		return l
	}
	return &Logger{l.Logger.With(fields...)}
}

// FromContext 辅助：若 ctx 中已经通过中间件放置 *Logger 则直接返回
func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return nil
	}
	if lg, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return lg
	}
	return nil
}

type loggerKey struct{}

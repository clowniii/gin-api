package observability

import (
	"context"
	"go-apiadmin/internal/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerContextMiddleware 将 trace_id / user_id 注入 logger，并放入请求 context
// 这样 handler 可以通过 logging.FromContext(c.Request.Context()) 直接获取带字段 logger
func LoggerContextMiddleware(base *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if v, ok := c.Get(TraceIDKey); ok {
			ctx = context.WithValue(ctx, "trace_id", v)
		}
		if uid, ok := c.Get("user_id"); ok {
			ctx = context.WithValue(ctx, "user_id", uid)
		}
		lg := base.WithContext(ctx)
		ctx = context.WithValue(ctx, loggerKey{}, lg)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// For internal context key reuse
type loggerKey struct{}

// Helper to log request ending (optional future extension)
func logRequestEnd(lg *logging.Logger, path string, status int) {
	if lg != nil {
		lg.Info("request_done", zap.String("path", path), zap.Int("status", status))
	}
}

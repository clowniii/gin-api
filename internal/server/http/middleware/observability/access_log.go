package observability

import (
	"time"

	"go-apiadmin/internal/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLog 输出基础 HTTP 访问日志：method, path, status, latency, ip
// 放置顺序建议在业务处理后（ResponseWrapper 之后）、Metrics 之前。
func AccessLog(l *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		l.Info("http_access",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

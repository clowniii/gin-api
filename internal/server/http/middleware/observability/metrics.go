package observability

import (
	"strconv"
	"time"

	"go-apiadmin/internal/metrics"

	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics.Inflight.Inc()
		start := time.Now()
		c.Next()
		metrics.Inflight.Dec()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		metrics.RequestDuration.WithLabelValues(path, c.Request.Method).Observe(time.Since(start).Seconds())
		metrics.RequestTotal.WithLabelValues(path, c.Request.Method, strconv.Itoa(c.Writer.Status())).Inc()
	}
}

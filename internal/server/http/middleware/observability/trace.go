package observability

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const TraceIDKey = "trace_id"

func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		// 提取上游上下文 (W3C traceparent / baggage)
		prop := otel.GetTextMapPropagator()
		ctx := prop.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		tr := otel.GetTracerProvider().Tracer("http-server")
		ctx, span := tr.Start(ctx, c.FullPath(), oteltrace.WithAttributes(attribute.String("custom.trace_id", traceID)))
		c.Set(TraceIDKey, traceID)
		c.Writer.Header().Set("X-Trace-Id", traceID)
		// 将当前 span context 注入响应头，便于下游继续
		prop.Inject(ctx, propagation.HeaderCarrier(c.Writer.Header()))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
		span.End()
	}
}

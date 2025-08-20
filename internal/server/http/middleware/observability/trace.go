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
		c.Set(TraceIDKey, traceID)
		c.Writer.Header().Set("X-Trace-Id", traceID)
		prop := otel.GetTextMapPropagator()
		ctx := prop.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		tr := otel.GetTracerProvider().Tracer("http-server")
		ctx, span := tr.Start(ctx, c.FullPath(), oteltrace.WithAttributes(attribute.String("custom.trace_id", traceID)))
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
	}
}

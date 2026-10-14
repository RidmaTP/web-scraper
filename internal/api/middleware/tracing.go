package middleware

import (
	"time"

	"github.com/RidmaTP/web-analyzer/internal/observability"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func ObservabilityMiddleware(metrics *observability.Metrics, tracer trace.Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := tracer.Start(c.Request.Context(), c.Request.Method+" "+c.FullPath())
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		attrs := []attribute.KeyValue{
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", status),
		}

		span.SetAttributes(attrs...)
		if status >= 500 {
			span.SetStatus(codes.Error, "server error")
		}

		metrics.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
		metrics.RequestDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
		if status >= 500 {
			metrics.ErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}
}

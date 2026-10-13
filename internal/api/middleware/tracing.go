package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

func TracingMiddleware(c *gin.Context) {
	tracer := otel.Tracer("web-analyzer/http")
	ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
	ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.Request.URL.Path)
	defer span.End()
	span.SetAttributes(attribute.String("http.request_id", c.GetHeader("X-Request-ID")))

	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

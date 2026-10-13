package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"runtime"
	"time"

	"github.com/RidmaTP/web-analyzer/internal/configs"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

func LogMiddleware(c *gin.Context) {
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)

	start := time.Now()
	c.Next()
	duration := time.Since(start)

	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)

	logger := configs.GetLogger()
	requestID := c.GetHeader("X-Request-ID")
	span := trace.SpanFromContext(c.Request.Context())
	logger.InfoContext(c.Request.Context(), "API request processed",
		"request_id", requestID,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", c.Writer.Status(),
		"duration_ms", duration.Milliseconds(),
		"time", start.Format(time.RFC3339),
		"ip", c.ClientIP(),
		"user_agent", c.Request.UserAgent(),
		"alloc_kb", (endMem.Alloc-startMem.Alloc)/1024,
		"total_alloc_kb", (endMem.TotalAlloc-startMem.TotalAlloc)/1024,
		"heap_objects", endMem.HeapObjects-startMem.HeapObjects,
		"num_gc", endMem.NumGC-startMem.NumGC,
		"trace_id", span.SpanContext().TraceID().String(),
		"span_id", span.SpanContext().SpanID().String(),
	)
}

func RequestIDMiddleware(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = newRequestID()
	}
	c.Request.Header.Set("X-Request-ID", requestID)
	c.Writer.Header().Set("X-Request-ID", requestID)
	c.Next()
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}

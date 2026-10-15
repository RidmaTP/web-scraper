package middleware

import (
	"runtime"
	"time"

	"github.com/RidmaTP/web-analyzer/internal/configs"
	"github.com/gin-gonic/gin"
)

func LogMiddleware(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	c.Request = c.Request.WithContext(configs.WithRequestID(c.Request.Context(), requestID))

	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)

	start := time.Now()
	c.Next()
	duration := time.Since(start)

	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)

	logger := configs.GetLogger()
	logger.InfoContext(c.Request.Context(), "API request processed",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", c.Writer.Status(),
		"duration", duration.String(),
		"time", start.Format(time.RFC3339),
		"ip", c.ClientIP(),
		"user-agent", c.Request.UserAgent(),
		"alloc_kb", (endMem.Alloc-startMem.Alloc)/1024,
		"total_alloc_kb", (endMem.TotalAlloc-startMem.TotalAlloc)/1024,
		"heap_objects", endMem.HeapObjects-startMem.HeapObjects,
		"num_gc", endMem.NumGC-startMem.NumGC,
		"x-request-id",requestID,
	)
}

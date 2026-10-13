package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "web_analyzer_http_requests_total", Help: "Total HTTP requests."},
		[]string{"method", "route", "status"},
	)
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "web_analyzer_http_request_duration_seconds", Help: "HTTP request duration in seconds."},
		[]string{"method", "route"},
	)
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration)
}

func MetricsMiddleware(c *gin.Context) {
	start := time.Now()
	c.Next()

	route := c.FullPath()
	if route == "" {
		route = "unknown"
	}
	status := strconv.Itoa(c.Writer.Status())
	httpRequests.WithLabelValues(c.Request.Method, route, status).Inc()
	httpDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
}

func MetricsHandler() gin.HandlerFunc {
	handler := promhttp.Handler()
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

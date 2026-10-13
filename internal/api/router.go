package api

import (
	"net/http"

	"github.com/RidmaTP/web-analyzer/internal/api/handlers"
	"github.com/RidmaTP/web-analyzer/internal/api/middleware"
	"github.com/RidmaTP/web-analyzer/internal/configs"
	"github.com/RidmaTP/web-analyzer/internal/observability"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// defines all the routes
func Router(engine *gin.Engine, metrics *observability.Metrics, tracer trace.Tracer) {
	engine.Use(cors.Default())

	group := engine.Group("/api/")
	group.Use(middleware.LogMiddleware)
	group.Use(middleware.ObservabilityMiddleware(metrics, tracer))
	group.Use(middleware.RateLimitMiddleware())
	routes(group, tracer)
}

func routes(rg *gin.RouterGroup, tracer trace.Tracer) {
	rg.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "service up and running",
			"status":  "success",
			"version": configs.GetAppVersion()})
	})

	rg.GET("/result", handlers.GetResultsHandler(tracer))

}

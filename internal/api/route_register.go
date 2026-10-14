package api

import (
	"github.com/RidmaTP/web-analyzer/internal/api/middleware"
	v1 "github.com/RidmaTP/web-analyzer/internal/api/v1"
	"github.com/RidmaTP/web-analyzer/internal/api/v1/handlers"
	"github.com/RidmaTP/web-analyzer/internal/observability"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/patrickmn/go-cache"
)

// defines all the routes
func RegisterRoutes(engine *gin.Engine, metrics *observability.Metrics, tracer trace.Tracer, cache *cache.Cache) {
	engine.Use(cors.Default())

	api := engine.Group("/api")
	api.Use(middleware.LogMiddleware)
	api.Use(middleware.ObservabilityMiddleware(metrics, tracer))
	api.Use(middleware.RateLimitMiddleware())

	v1.RegisterV1Routes(api.Group("/v1"), handlers.NewResultsHandler(cache, tracer))

}

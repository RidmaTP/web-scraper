package handlers

import (
	"fmt"

	"github.com/RidmaTP/web-analyzer/internal/application"
	"github.com/RidmaTP/web-analyzer/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel/trace"
)

type ResultsHandler struct {
	analysis *application.AnalysisConfig
}

func NewResultsHandler(cache *cache.Cache, tracer trace.Tracer) *ResultsHandler {
	return &ResultsHandler{
		analysis: &application.AnalysisConfig{
			Cache:  cache,
			Tracer: tracer,
		},
	}
}

func (h *ResultsHandler) GetResults(c *gin.Context) {
	targetURL := c.Query("url")
	ctx := c.Request.Context()

	if errObj := h.analysis.ValidateURL(ctx, &targetURL); errObj != nil {
		c.AbortWithStatusJSON(errObj.StatusCode, gin.H{
			"error":       errObj.Error,
			"status_code": errObj.StatusCode,
		})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	session, _ := h.analysis.Start(ctx, targetURL)
	for {
		select {
		case errObj, ok := <-session.Errors:
			if !ok {
				session.Errors = nil
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", *utils.ErrStreamObj(*errObj))
			c.Writer.Flush()
			return
		case <-ctx.Done():
			return
		case message, ok := <-session.EventStream:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", message)
			c.Writer.Flush()
		}
	}
}

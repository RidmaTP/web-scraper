package handlers

import (
	"fmt"
	"time"

	"github.com/RidmaTP/web-analyzer/internal/configs"
	"github.com/RidmaTP/web-analyzer/internal/models"
	"github.com/RidmaTP/web-analyzer/internal/services/analyzer"
	"github.com/RidmaTP/web-analyzer/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel/trace"
)

// Gin Api handler used to get a url
// sends a text/event-stream in http1.1
func GetResultsHandler(tracer trace.Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {

		url := c.Query("url")

		ctx := c.Request.Context()

		errObj := utils.ValidateRequestURL(ctx, &url)
		if errObj != nil {
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

		a := analyzer.NewBodyAnalyzer(tracer)
		fmt.Println(url)

		//checking cache for results for the given url
		cacheObj := configs.GetCacheConfig()

		if cachedData, found := cacheObj.Get(url); found {
			a := cachedData.(string)
			fmt.Fprintf(c.Writer, "data: %s\n\n", a)
			c.Writer.Flush()
			return
		}

		errChan := make(chan *models.ErrorOut, 1)

		go func(cache *cache.Cache) {
			defer close(a.Stream)
			errObj := a.Analyze(ctx, url)
			if errObj != nil {
				errChan <- errObj
				return
			}
			if strObj, err := utils.JsonToText(a.Output); err == nil {
				cache.Set(url, *strObj, 2*time.Hour)
			}

		}(cacheObj)
		for {
			select {
			//used to push the error
			case errObj := <-errChan:
				if errObj != nil {
					fmt.Fprintf(c.Writer, "data: %s\n\n", *utils.ErrStreamObj(*errObj))
					c.Writer.Flush()

					return
				}
			case <-ctx.Done():
				fmt.Println("client disconnected")
				return
			case msg, ok := <-a.Stream:
				if !ok {
					return
				}
				fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
				c.Writer.Flush()
			}
		}
	}
}

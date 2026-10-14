package v1

import (
	"net/http"

	"github.com/RidmaTP/web-analyzer/internal/api/v1/handlers"
	"github.com/RidmaTP/web-analyzer/internal/configs"
	"github.com/gin-gonic/gin"
)

func RegisterV1Routes(group *gin.RouterGroup,resultHandler *handlers.ResultsHandler) {
	group.GET("/health", health)
	group.GET("/result", resultHandler.GetResults)
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "service up and running",
		"status":  "success",
		"version": configs.GetAppVersion(),
	})
}

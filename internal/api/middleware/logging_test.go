package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddlewarePreservesAndGeneratesIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware)
	router.GET("/", func(c *gin.Context) {
		c.Status(204)
	})

	t.Run("preserves incoming request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", "client-request-id")
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		if got := recorder.Header().Get("X-Request-ID"); got != "client-request-id" {
			t.Fatalf("request ID = %q, want client-request-id", got)
		}
	})

	t.Run("generates missing request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		if got := recorder.Header().Get("X-Request-ID"); got == "" {
			t.Fatal("expected generated request ID")
		}
	})
}

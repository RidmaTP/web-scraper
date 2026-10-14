package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RidmaTP/web-analyzer/internal/api"
	"github.com/RidmaTP/web-analyzer/internal/api/middleware"
	"github.com/RidmaTP/web-analyzer/internal/browser"
	"github.com/RidmaTP/web-analyzer/internal/configs"
	"github.com/RidmaTP/web-analyzer/internal/observability"
	"github.com/gin-gonic/gin"
)

// initiating gin server on default port 8000
// port can be changed from configs/.env
func main() {
	app := gin.Default()
	go middleware.CleanupUsers()

	err := configs.LoadEnv()
	if err != nil {
		log.Fatal(err)
	}
	configs.LoadLogger()
	configs.LoadCacheConfig()

	ctx := context.Background()
	meterProvider, metrics, tracerProvider, err := observability.Initialize(ctx)
	if err != nil {
		log.Fatalf("failed to initialize observability: %v", err)
	}

	tracer := tracerProvider.Tracer("web-analyzer")

	api.RegisterRoutes(app, metrics, tracer, configs.GetCacheConfig())

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := meterProvider.Shutdown(shutdownCtx); err != nil {
			log.Printf("failed to shut down meter provider: %v", err)
		}
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			log.Printf("failed to shut down tracer provider: %v", err)
		}
	}()
	if err := browser.LaunchBrowser(configs.GetChromePath()); err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    ":" + configs.GetPort(),
		Handler: app,
	}

	go func() {
		log.Printf("server listening on %s", server.Addr)
		server.ListenAndServe()
	}()

	//graceful shutdown

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	sig := <-shutdownSignal
	log.Printf("received %s, shutting down", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}

	if err := browser.CloseBrowser(); err != nil {
		log.Printf("unable to close rod browser: %v", err)
	}
}

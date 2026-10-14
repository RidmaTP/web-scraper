package application

import (
	"context"
	"time"

	"github.com/RidmaTP/web-analyzer/internal/analysis"
	"github.com/RidmaTP/web-analyzer/internal/models"
	"github.com/RidmaTP/web-analyzer/internal/utils"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel/trace"
)

const resultCacheDuration = 2 * time.Hour

type AnalysisSession struct {
	EventStream <-chan string
	Errors      <-chan *models.ErrorOut
}

type AnalysisConfig struct {
	Cache  *cache.Cache
	Tracer trace.Tracer
}

func NewAnalysisService(cacheStore *cache.Cache, tracer trace.Tracer) *AnalysisConfig {
	return &AnalysisConfig{Cache: cacheStore, Tracer: tracer}
}

func (s *AnalysisConfig) ValidateURL(ctx context.Context, targetURL *string) *models.ErrorOut {
	return utils.ValidateRequestURL(ctx, targetURL)
}

func (s *AnalysisConfig) Start(ctx context.Context, targetURL string) (*AnalysisSession, bool) {
	if cached, found := s.Cache.Get(targetURL); found {
		if result, ok := cached.(string); ok {
			events := make(chan string, 1)
			events <- result
			close(events)
			return &AnalysisSession{EventStream: events}, true
		}
	}

	analyzer := analysis.NewBodyAnalyzer(s.Tracer)
	errors := make(chan *models.ErrorOut, 1)

	go func() {
		defer close(analyzer.Stream)
		if errObj := analyzer.Analyze(ctx, targetURL); errObj != nil {
			errors <- errObj
			close(errors)
			return
		}

		result, err := utils.JsonToText(analyzer.Output)
		if err != nil {
			errors <- &models.ErrorOut{
				StatusCode: 500,
				Error:      err.Error(),
			}
			close(errors)
			return
		}
		s.Cache.Set(targetURL, *result, resultCacheDuration)
		close(errors)
	}()

	return &AnalysisSession{
		EventStream: analyzer.Stream,
		Errors:      errors,
	}, false
}

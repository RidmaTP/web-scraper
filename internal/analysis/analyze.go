package analysis

import (
	"context"
	"io"
	"net/http"
	"runtime"
	"sync"

	"github.com/RidmaTP/web-analyzer/internal/browser"
	"github.com/RidmaTP/web-analyzer/internal/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/html"
)

type BodyFetcher interface {
	FetchBody(ctx context.Context, url string) (io.ReadCloser, error)
	FetchLink(ctx context.Context, url string) error
}

func (a *BodyAnalyzer) fetchBody(ctx context.Context, url string) (io.ReadCloser, error) {
	return a.Fetcher.FetchBody(ctx, url)
}

func (a *BodyAnalyzer) checkLink(ctx context.Context, url string) error {
	return a.Fetcher.FetchLink(ctx, url)
}

type BodyAnalyzer struct {
	Fetcher         BodyFetcher
	Stream          chan string
	Output          models.Output
	muOutput        sync.Mutex
	muActiveLinks   sync.Mutex
	muInactiveLinks sync.Mutex
	wg              *sync.WaitGroup
	Workers         int
	ctx             context.Context
	tracer          trace.Tracer
}

func (a *BodyAnalyzer) getTracer() trace.Tracer {
	if a.tracer == nil {
		a.tracer = otel.Tracer("web-analyzer")
	}
	return a.tracer
}

func NewBodyAnalyzer(tracers ...trace.Tracer) *BodyAnalyzer {
	tracer := otel.Tracer("web-analyzer")
	if len(tracers) > 0 && tracers[0] != nil {
		tracer = tracers[0]
	}
	return &BodyAnalyzer{
		Fetcher: &browser.Fetcher{},
		Stream:  make(chan string, 20),
		Output:  models.Output{},
		Workers: runtime.NumCPU(),
		tracer:  tracer,
	}
}

// Analyze runs the analysis and supports cancellation through the request context.
func (a *BodyAnalyzer) Analyze(ctx context.Context, url string) *models.ErrorOut {
	tracer := a.getTracer()
	ctx, span := tracer.Start(ctx, "analyze page")
	defer span.End()
	span.SetAttributes(attribute.String("url.full", url))

	var inTitle bool
	a.wg = &sync.WaitGroup{}
	a.ctx = ctx
	workers := a.Workers
	if workers <= 0 {
		workers = 1
	}
	linkJobQueue := make(chan string, workers)
	loginFlags := models.LoginFlags{}

	fetchCtx, fetchSpan := tracer.Start(ctx, "fetch page")
	ioReader, err := a.fetchBody(fetchCtx, url)
	fetchSpan.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "page fetch failed")
		return &models.ErrorOut{StatusCode: http.StatusBadGateway, Error: err.Error()}
	}
	defer ioReader.Close()
	defer func() {
		close(linkJobQueue)
		a.wg.Wait()
	}()
	_, parseSpan := tracer.Start(ctx, "parse HTML")
	defer parseSpan.End()
	tokenizer := html.NewTokenizer(ioReader)

	for i := 0; i < workers; i++ {
		a.wg.Add(1)
		go func(a *BodyAnalyzer, linkJobQueue chan string, baseURL string) {
			defer a.wg.Done()
			a.ActiveCheckWorkerContext(ctx, baseURL, linkJobQueue)
		}(a, linkJobQueue, url)
	}

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			parseSpan.RecordError(err)
			parseSpan.SetStatus(codes.Error, "HTML parsing failed")
			return &models.ErrorOut{StatusCode: http.StatusInternalServerError, Error: err.Error()}
		}
		token := tokenizer.Token()

		isInTitle, err := a.FindTitle(tokenType, token, inTitle)
		if err != nil {
			return &models.ErrorOut{StatusCode: http.StatusInternalServerError, Error: err.Error()}
		}
		inTitle = isInTitle

		if err = a.FindHTMLVersion(tokenType, token); err != nil {
			return &models.ErrorOut{StatusCode: http.StatusInternalServerError, Error: err.Error()}
		}
		if err = a.FindHeaderCount(tokenType, token); err != nil {
			return &models.ErrorOut{StatusCode: http.StatusInternalServerError, Error: err.Error()}
		}
		if err = a.FindLinks(tokenType, token, url, &linkJobQueue); err != nil {
			return &models.ErrorOut{StatusCode: http.StatusInternalServerError, Error: err.Error()}
		}
		if err = a.FindIfLogin(tokenType, token, &loginFlags); err != nil {
			return &models.ErrorOut{StatusCode: http.StatusInternalServerError, Error: err.Error()}
		}
	}

	a.muOutput.Lock()
	defer a.muOutput.Unlock()
	if a.Output.Headers == nil {
		a.Output.Headers = make(map[string]int)
	}
	if err := a.emitLocked(); err != nil {
		return &models.ErrorOut{StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	return nil
}

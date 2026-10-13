package analyzer

import (
	"context"
	"io"
	"net/http"
	"runtime"
	"sync"

	"github.com/RidmaTP/web-analyzer/internal/models"
	"github.com/RidmaTP/web-analyzer/internal/services/fetcher"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/net/html"
)

type BodyFetcher interface {
	FetchBody(ctx context.Context, url string) (io.ReadCloser, error)
	FetchLink(ctx context.Context, url string) error
}

func (a *BodyAnalyzer) fetchBody(ctx context.Context, url string) (io.ReadCloser, error) {
	ctx, span := otel.Tracer("web-analyzer/analyzer").Start(ctx, "analyzer.fetch_body")
	defer span.End()
	body, err := a.Fetcher.FetchBody(ctx, url)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "body fetch failed")
	}
	return body, err
}

func (a *BodyAnalyzer) checkLink(ctx context.Context, url string) error {
	ctx, span := otel.Tracer("web-analyzer/analyzer").Start(ctx, "analyzer.check_link")
	defer span.End()
	err := a.Fetcher.FetchLink(ctx, url)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "link check failed")
	}
	return err
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
}

func NewBodyAnalyzer() *BodyAnalyzer {
	return &BodyAnalyzer{
		Fetcher: &fetcher.Fetcher{},
		Stream:  make(chan string, 20),
		Output:  models.Output{},
		Workers: runtime.NumCPU(),
	}
}

// Analyze runs the analysis and supports cancellation through the request context.
func (a *BodyAnalyzer) Analyze(ctx context.Context, url string) *models.ErrorOut {
	ctx, span := otel.Tracer("web-analyzer/analyzer").Start(ctx, "analyzer.analyze")
	defer span.End()
	ctx, parseSpan := otel.Tracer("web-analyzer/analyzer").Start(ctx, "analyzer.parse_html")
	defer parseSpan.End()
	var inTitle bool
	a.wg = &sync.WaitGroup{}
	a.ctx = ctx
	workers := a.Workers
	if workers <= 0 {
		workers = 1
	}
	linkJobQueue := make(chan string, workers)
	loginFlags := models.LoginFlags{}

	ioReader, err := a.fetchBody(ctx, url)
	if err != nil {
		return &models.ErrorOut{StatusCode: http.StatusBadGateway, Error: err.Error()}
	}
	defer ioReader.Close()
	defer func() {
		close(linkJobQueue)
		a.wg.Wait()
	}()
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
	return nil
}

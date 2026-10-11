package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Fetcher struct {
	BrowserTimeout time.Duration
	Client         *http.Client
}

// FetchBody loads the page in a headless browser and returns the rendered HTML.
func (f *Fetcher) FetchBody(ctx context.Context, url string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	body, err := f.fetchHTMLBodyRod(ctx, url)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader([]byte(*body))), nil
}

func (f *Fetcher) FetchLink(ctx context.Context, url string) error {
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("link returned status %d", resp.StatusCode)
	}
	return nil
}

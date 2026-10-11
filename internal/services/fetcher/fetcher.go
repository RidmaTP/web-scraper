package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetcher contract

type Fetcher struct {
	Client *http.Client
}

// // Returns the reader to read the body
// func (f *Fetcher) FetchBody(url string) (io.ReadCloser, error) {
// 	return f.FetchBodyContext(context.Background(), url)
// }

// FetchBodyContext performs a bounded, cancellable request.
func (f *Fetcher) FetchBody(ctx context.Context, url string) (io.ReadCloser, error) {
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%d is returned", resp.StatusCode)
	}
	return resp.Body, nil
}

package browser

import (
	"context"
	"strings"
	"testing"
)

func TestFetcherHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (&Fetcher{}).FetchBody(ctx, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("FetchBody() expected cancellation error")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("FetchBody() error = %v, want context cancellation", err)
	}
}

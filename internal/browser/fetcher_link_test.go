package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetcherFetchLink(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "successful response", statusCode: http.StatusOK},
		{name: "non-success response", statusCode: http.StatusNotFound, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			err := (&Fetcher{}).FetchLink(context.Background(), server.URL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FetchLink() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetcherFetchLinkRejectsInvalidURL(t *testing.T) {
	err := (&Fetcher{}).FetchLink(context.Background(), "://invalid")
	if err == nil {
		t.Fatal("FetchLink() expected an error for an invalid URL")
	}
}

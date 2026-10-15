package application

import (
	"context"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
)

func TestAnalysisConfigStartReturnsCachedResult(t *testing.T) {
	const targetURL = "https://example.com"
	const cachedResult = `{"title":"cached"}`

	store := cache.New(time.Hour, time.Minute)
	store.Set(targetURL, cachedResult, cache.DefaultExpiration)
	service := NewAnalysisService(store, nil)

	session, cached := service.Start(context.Background(), targetURL)
	if !cached {
		t.Fatal("Start() cached = false, want true")
	}

	result, ok := <-session.EventStream
	if !ok {
		t.Fatal("EventStream closed before returning cached result")
	}
	if result != cachedResult {
		t.Fatalf("cached result = %q, want %q", result, cachedResult)
	}

	if _, ok := <-session.EventStream; ok {
		t.Fatal("EventStream should be closed after cached result")
	}
}

func TestAnalysisConfigValidateURL(t *testing.T) {
	service := NewAnalysisService(cache.New(time.Hour, time.Minute), nil)

	if err := service.ValidateURL(context.Background(), stringPtr("https://example.com")); err != nil {
		t.Fatalf("ValidateURL() unexpected error = %v", err)
	}

	err := service.ValidateURL(context.Background(), stringPtr("http://"))
	if err == nil {
		t.Fatal("ValidateURL() expected an error for an invalid URL")
	}
}

func stringPtr(value string) *string {
	return &value
}

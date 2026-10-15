package configs

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestRequestIDHandlerAddsRequestID(t *testing.T) {
	var output bytes.Buffer
	handler := requestIDHandler{
		Handler: slog.NewJSONHandler(&output, nil),
	}
	logger := slog.New(handler)

	logger.InfoContext(WithRequestID(context.Background(), "request-123"), "request completed")

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode log record: %v", err)
	}
	if got := record["request_id"]; got != "request-123" {
		t.Fatalf("request_id = %v, want request-123", got)
	}
}

func TestRequestIDHandlerOmitsEmptyRequestID(t *testing.T) {
	var output bytes.Buffer
	handler := requestIDHandler{
		Handler: slog.NewJSONHandler(&output, nil),
	}

	slog.New(handler).InfoContext(context.Background(), "request completed")

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode log record: %v", err)
	}
	if _, ok := record["request_id"]; ok {
		t.Fatal("request_id should be omitted when it is not in the context")
	}
}

func TestRequestIDHandlerPreservesAttrsAndGroups(t *testing.T) {
	var output bytes.Buffer
	handler := requestIDHandler{
		Handler: slog.NewJSONHandler(&output, nil),
	}

	slog.New(handler).
		WithGroup("request").
		With("component", "api").
		InfoContext(WithRequestID(context.Background(), "request-456"), "request completed")

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode log record: %v", err)
	}
	request, ok := record["request"].(map[string]any)
	if !ok {
		t.Fatalf("request group = %v, want an object", record["request"])
	}
	if request["request_id"] != "request-456" {
		t.Fatalf("grouped request_id = %v, want request-456", request["request_id"])
	}
	if request["component"] != "api" {
		t.Fatalf("grouped component = %v, want api", request["component"])
	}
}

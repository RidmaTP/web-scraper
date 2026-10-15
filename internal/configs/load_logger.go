package configs

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

var (
	loadLoggerOnce sync.Once
	logger         *slog.Logger
)

type requestIDContextKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok && requestID != ""
}

type requestIDHandler struct {
	slog.Handler
}

func (h requestIDHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID, ok := requestIDFromContext(ctx); ok {
		record.AddAttrs(slog.String("request_id", requestID))
	}
	return h.Handler.Handle(ctx, record)
}

func (h requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestIDHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h requestIDHandler) WithGroup(name string) slog.Handler {
	return requestIDHandler{Handler: h.Handler.WithGroup(name)}
}

func LoadLogger() *slog.Logger {
	loadLoggerOnce.Do(func() {
		logger = slog.New(requestIDHandler{
			Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: false,
			}),
		})
		slog.SetDefault(logger)
	})
	return logger
}

func GetLogger() *slog.Logger {
	return LoadLogger()
}

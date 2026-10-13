package configs

import (
	"log/slog"
	"os"
	"sync"
)

var (
	loadLoggerOnce sync.Once
	logger         *slog.Logger
)

func LoadLogger() *slog.Logger {
	loadLoggerOnce.Do(func() {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	})
	return logger
}

func GetLogger() *slog.Logger {
	if logger == nil {
		LoadLogger()
	}
	return logger
}

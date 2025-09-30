package logger

import (
	"log/slog"
	"os"
	"web-server/internal/config"
)

type Logger struct {
	*slog.Logger
}

func New(cfg *config.Config) *Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: false})
	lg := slog.New(handler)
	return &Logger{lg}
}

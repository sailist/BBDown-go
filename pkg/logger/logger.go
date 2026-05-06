package logger

import (
	"log/slog"
	"os"
)

func New(level slog.Level, format Format) *slog.Logger {
	handler := NewHandler(os.Stderr, level, format)
	return slog.New(handler)
}

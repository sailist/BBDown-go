package logger

import (
	slog "golang.org/x/exp/slog"
	"os"
)

func New(level slog.Level, format Format) *slog.Logger {
	handler := NewHandler(os.Stderr, level, format)
	return slog.New(handler)
}

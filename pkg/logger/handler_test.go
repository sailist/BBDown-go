package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, slog.LevelInfo, FormatText)
	logger := slog.New(handler)

	logger.Debug("debug msg")
	if buf.Len() != 0 {
		t.Errorf("expected no output for DEBUG when level is INFO, got: %s", buf.String())
	}

	buf.Reset()
	logger.Info("info msg")
	if !strings.Contains(buf.String(), "info msg") {
		t.Errorf("expected INFO msg to be logged, got: %s", buf.String())
	}
}

func TestAttributeInjection(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, slog.LevelDebug, FormatText)
	logger := slog.New(handler).With("key", "value")

	logger.Info("msg")
	if !strings.Contains(buf.String(), "key=value") {
		t.Errorf("expected attribute in output, got: %s", buf.String())
	}
}

func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, slog.LevelDebug, FormatJSON)
	logger := slog.New(handler).With("key", "value")

	logger.Info("hello")

	var result map[string]interface{}
	line := strings.TrimSpace(buf.String())
	if err := json.Unmarshal([]byte(line), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v, output: %s", err, buf.String())
	}

	if result["msg"] != "hello" {
		t.Errorf("expected msg=hello, got: %v", result["msg"])
	}
	if result["key"] != "value" {
		t.Errorf("expected key=value, got: %v", result["key"])
	}
	if result["level"] != "INFO" {
		t.Errorf("expected level=INFO, got: %v", result["level"])
	}
	if _, ok := result["time"]; !ok {
		t.Errorf("expected time field")
	}
}

func TestTextFormatColors(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, slog.LevelDebug, FormatText)
	logger := slog.New(handler)

	logger.Info("test")
	output := buf.String()
	if !strings.Contains(output, colorGreen) {
		t.Errorf("expected green color code for INFO level, got: %s", output)
	}
	if !strings.Contains(output, colorGray) {
		t.Errorf("expected gray color code for time, got: %s", output)
	}
	if !strings.Contains(output, colorReset) {
		t.Errorf("expected reset color code, got: %s", output)
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, slog.LevelDebug, FormatText)
	logger := slog.New(handler).WithGroup("group").With("key", "value")

	logger.Info("msg")
	if !strings.Contains(buf.String(), "group.key=value") {
		t.Errorf("expected grouped attribute in output, got: %s", buf.String())
	}
}

func TestEnabled(t *testing.T) {
	var buf bytes.Buffer
	handler := NewHandler(&buf, slog.LevelWarn, FormatText)

	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected DEBUG to be disabled when level is WARN")
	}
	if handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected INFO to be disabled when level is WARN")
	}
	if !handler.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected WARN to be enabled when level is WARN")
	}
	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Error("expected ERROR to be enabled when level is WARN")
	}
}

package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type Format int

const (
	FormatText Format = iota
	FormatJSON
)

const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

var levelColors = map[slog.Level]string{
	slog.LevelDebug: colorCyan,
	slog.LevelInfo:  colorGreen,
	slog.LevelWarn:  colorYellow,
	slog.LevelError: colorRed,
}

var levelNames = map[slog.Level]string{
	slog.LevelDebug: "DEBUG",
	slog.LevelInfo:  "INFO",
	slog.LevelWarn:  "WARN",
	slog.LevelError: "ERROR",
}

type Handler struct {
	level  slog.Leveler
	format Format
	attrs  []slog.Attr
	groups []string
	out    io.Writer
	mu     sync.Mutex
}

func NewHandler(out io.Writer, level slog.Leveler, format Format) *Handler {
	return &Handler{
		level:  level,
		format: format,
		out:    out,
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	switch h.format {
	case FormatJSON:
		return h.handleJSON(r)
	default:
		return h.handleText(r)
	}
}

func (h *Handler) handleText(r slog.Record) error {
	var sb strings.Builder

	// Time in gray
	sb.WriteString(colorGray)
	sb.WriteString(r.Time.Format("2006-01-02 15:04:05"))
	sb.WriteString(colorReset)
	sb.WriteString("  ")

	// Level in color
	color := levelColors[r.Level]
	sb.WriteString(color)
	sb.WriteString(fmt.Sprintf("%-5s", levelNames[r.Level]))
	sb.WriteString(colorReset)
	sb.WriteString("  ")

	// Message
	sb.WriteString(r.Message)

	// Attributes
	prefix := h.groupPrefix()
	for _, attr := range h.attrs {
		sb.WriteString("  ")
		sb.WriteString(prefix)
		sb.WriteString(attr.Key)
		sb.WriteString("=")
		sb.WriteString(attr.Value.String())
	}
	r.Attrs(func(a slog.Attr) bool {
		sb.WriteString("  ")
		sb.WriteString(prefix)
		sb.WriteString(a.Key)
		sb.WriteString("=")
		sb.WriteString(a.Value.String())
		return true
	})

	sb.WriteString("\n")

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write([]byte(sb.String()))
	return err
}

func (h *Handler) handleJSON(r slog.Record) error {
	m := make(map[string]interface{})
	m["time"] = r.Time.Format(time.RFC3339)
	m["level"] = r.Level.String()
	m["msg"] = r.Message

	// Add handler attrs with group nesting
	h.addAttrsToMap(m, h.attrs)

	// Add record attrs with group nesting
	r.Attrs(func(a slog.Attr) bool {
		h.addAttrsToMap(m, []slog.Attr{a})
		return true
	})

	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err = h.out.Write(append(b, '\n'))
	return err
}

func (h *Handler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

func (h *Handler) addAttrsToMap(m map[string]interface{}, attrs []slog.Attr) {
	for _, attr := range attrs {
		current := m
		for _, g := range h.groups {
			if next, ok := current[g].(map[string]interface{}); ok {
				current = next
			} else {
				next := make(map[string]interface{})
				current[g] = next
				current = next
			}
		}
		current[attr.Key] = attr.Value.Any()
	}
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &Handler{
		level:  h.level,
		format: h.format,
		attrs:  newAttrs,
		groups: h.groups,
		out:    h.out,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &Handler{
		level:  h.level,
		format: h.format,
		attrs:  h.attrs,
		groups: newGroups,
		out:    h.out,
	}
}

package logger

import (
	"io"
	"log/slog"
)

const (
	FormatJSON = "json"
	FormatText = "text"
)

func ValidFormat(f string) bool { return f == FormatJSON || f == FormatText }

func New(w io.Writer, level slog.Level, format, service string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	switch format {
	case FormatText:
		h = slog.NewTextHandler(w, opts)
	default:
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h).With("service", service)
}

package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/coreos/go-systemd/v22/journal"
)

// JournaldHandler is a custom slog.Handler that sends logs to journald.
type JournaldHandler struct{}

// Enabled always returns true to enable logging for all levels.
func (h *JournaldHandler) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

// Handle sends the log record to journald.
func (h *JournaldHandler) Handle(_ context.Context, r slog.Record) error {
	// Format the log message
	msg := r.Message
	attrs := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		attrs[strings.ToUpper(a.Key)] = a.Value.String()
		msg += fmt.Sprintf(", %s=%s", a.Key, a.Value.String())
		return true
	})

	// Map slog levels to journald priority levels
	var priority journal.Priority
	switch r.Level {
	case slog.LevelDebug:
		priority = journal.PriDebug
	case slog.LevelInfo:
		priority = journal.PriInfo
	case slog.LevelWarn:
		priority = journal.PriWarning
	case slog.LevelError:
		priority = journal.PriErr
	default:
		priority = journal.PriNotice
	}

	// Send the log to journald
	return journal.Send(msg, priority, attrs)
}

// WithAttrs returns a new handler with the given attributes.
func (h *JournaldHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup returns a new handler with the given group name.
func (h *JournaldHandler) WithGroup(name string) slog.Handler {
	return h
}

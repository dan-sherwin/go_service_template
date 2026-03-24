package app

import (
	"context"
	"log/slog"
	"os"
	"os/user"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
)

type TeeHandler struct {
	handlers []slog.Handler
}

func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &TeeHandler{handlers: newHandlers}
}

func (h *TeeHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &TeeHandler{handlers: newHandlers}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func setDefaultLogger(base *slog.Logger) {
	user, _ := user.Current()
	child := base.With(
		slog.Int("pid", os.Getpid()),
		slog.String("user", user.Username),
		slog.String("app", consts.APPNAME),
		slog.String("version", consts.Version),
		slog.String("commit", consts.Commit),
		slog.String("buildDate", consts.BuildDate),
	)
	slog.SetDefault(child)
}

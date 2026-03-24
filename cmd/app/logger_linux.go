package app

import (
	"log/slog"
	"os"
)

var LoggingLevel = "debug"

func initLogger() {
	lvl := parseLevel(LoggingLevel)
	journaldMinLevel = lvl
	var handlers []slog.Handler
	handlers = append(handlers, &JournaldHandler{})
	if cliConfig.Verbose {
		handlers = append(handlers, slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	}
	var handler slog.Handler
	if len(handlers) == 1 {
		handler = handlers[0]
	} else {
		handler = &TeeHandler{handlers: handlers}
	}
	logger := slog.New(handler)
	setDefaultLogger(logger)
}

package app

import (
	"log/slog"
	"os"
)

var LoggingLevel = "debug"

func initLogger() {
	lvl := parseLevel(LoggingLevel)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	// On macOS, it already logs to stdout by default, so -v is always satisfied.
	// But we can still use TeeHandler if we wanted more outputs in the future.
	logger := slog.New(handler)
	setDefaultLogger(logger)
}

package log

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON slog.Logger honoring OBS_LOG_LEVEL (debug|info|warn|error).
func New(service string) *slog.Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(os.Getenv("OBS_LOG_LEVEL")) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h).With("service", service)
}

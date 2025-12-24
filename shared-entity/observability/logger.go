package observability

import (
	"log/slog"
	"os"
)

// InitLogger initializes the global logger with a JSON logger.
// It sets the default slog logger to write structured JSON to stdout.
func InitLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

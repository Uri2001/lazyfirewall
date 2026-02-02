package logger

import (
	"log/slog"
	"os"
)

func Init() {
	level := slog.LevelInfo
	if os.Getenv("LAZYFIREWALL_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

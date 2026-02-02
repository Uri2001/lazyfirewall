package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var logFile *os.File

func Init() {
	level := slog.LevelInfo
	if os.Getenv("LAZYFIREWALL_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	writer := io.Writer(os.Stderr)
	if os.Getenv("LAZYFIREWALL_LOG_STDERR") != "1" {
		if path := resolveLogPath(); path != "" {
			if file, err := openLogFile(path); err == nil {
				logFile = file
				writer = file
			}
		}
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

func resolveLogPath() string {
	if path := os.Getenv("LAZYFIREWALL_LOG_FILE"); path != "" {
		return path
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "lazyfirewall", "lazyfirewall.log")
}

func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}

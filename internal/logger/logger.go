package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var logFile *os.File

func Init(levelOverride string) error {
	level := slog.LevelInfo
	if os.Getenv("LAZYFIREWALL_DEBUG") == "1" {
		level = slog.LevelDebug
	}
	if levelOverride != "" {
		parsed, err := ParseLevel(levelOverride)
		if err != nil {
			return err
		}
		level = parsed
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
	return nil
}

func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	case "":
		return slog.LevelInfo, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %q (use debug|info|warn|error)", value)
	}
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

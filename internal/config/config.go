package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	UI       UIConfig
	Behavior BehaviorConfig
	Advanced AdvancedConfig
}

type UIConfig struct {
	Theme string
}

type BehaviorConfig struct {
	DefaultPermanent   bool
	AutoRefreshSeconds int
}

type AdvancedConfig struct {
	LogLevel string
}

func Default() Config {
	return Config{
		UI: UIConfig{
			Theme: "default",
		},
		Behavior: BehaviorConfig{
			DefaultPermanent:   false,
			AutoRefreshSeconds: 0,
		},
		Advanced: AdvancedConfig{
			LogLevel: "",
		},
	}
}

func ResolvePath() (string, error) {
	if env := os.Getenv("LAZYFIREWALL_CONFIG"); env != "" {
		return env, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazyfirewall", "config.toml"), nil
}

func Load() (Config, []string, string, bool, error) {
	paths, err := candidatePaths()
	if err != nil {
		return Default(), nil, "", false, err
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Default(), nil, "", false, err
		}
		cfg := Default()
		warnings, err := parse(string(data), &cfg)
		if err != nil {
			return Default(), nil, "", false, fmt.Errorf("parse %s: %w", path, err)
		}
		warnings = append(warnings, normalizeConfig(&cfg)...)
		return cfg, warnings, path, true, nil
	}
	return Default(), nil, "", false, nil
}

func normalizeConfig(cfg *Config) []string {
	warnings := make([]string, 0)
	if cfg.UI.Theme != "" && cfg.UI.Theme != "default" {
		warnings = append(warnings, fmt.Sprintf("ui.theme %q is not supported; using default", cfg.UI.Theme))
		cfg.UI.Theme = "default"
	}
	if cfg.Behavior.AutoRefreshSeconds > 0 {
		warnings = append(warnings, "behavior.auto_refresh_interval is currently disabled; set to 0")
	}
	return warnings
}

func parse(raw string, cfg *Config) ([]string, error) {
	section := ""
	warnings := make([]string, 0)
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		line = stripComment(line)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return warnings, fmt.Errorf("line %d: expected key = value", i+1)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key == "" {
			return warnings, fmt.Errorf("line %d: empty key", i+1)
		}

		switch section {
		case "ui":
			if key == "theme" {
				val, err := parseString(value)
				if err != nil {
					return warnings, fmt.Errorf("line %d: %w", i+1, err)
				}
				cfg.UI.Theme = val
			} else {
				warnings = append(warnings, fmt.Sprintf("line %d: unknown ui key %q", i+1, key))
			}
		case "behavior":
			switch key {
			case "default_permanent":
				val, err := parseBool(value)
				if err != nil {
					return warnings, fmt.Errorf("line %d: %w", i+1, err)
				}
				cfg.Behavior.DefaultPermanent = val
			case "auto_refresh_interval":
				val, err := parseInt(value)
				if err != nil {
					return warnings, fmt.Errorf("line %d: %w", i+1, err)
				}
				cfg.Behavior.AutoRefreshSeconds = val
			default:
				warnings = append(warnings, fmt.Sprintf("line %d: unknown behavior key %q", i+1, key))
			}
		case "advanced":
			if key == "log_level" {
				val, err := parseString(value)
				if err != nil {
					return warnings, fmt.Errorf("line %d: %w", i+1, err)
				}
				cfg.Advanced.LogLevel = val
			} else {
				warnings = append(warnings, fmt.Sprintf("line %d: unknown advanced key %q", i+1, key))
			}
		default:
			warnings = append(warnings, fmt.Sprintf("line %d: unknown section %q", i+1, section))
		}
	}
	return warnings, nil
}

func candidatePaths() ([]string, error) {
	if env := os.Getenv("LAZYFIREWALL_CONFIG"); env != "" {
		return []string{env}, nil
	}
	primary, err := ResolvePath()
	if err != nil {
		return nil, err
	}
	paths := []string{primary}
	if sudoPath, ok := sudoConfigPath(primary); ok {
		paths = append(paths, sudoPath)
	}
	return paths, nil
}

func sudoConfigPath(primary string) (string, bool) {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		return "", false
	}
	current := os.Getenv("USER")
	if current == sudoUser {
		return "", false
	}
	u, err := user.Lookup(sudoUser)
	if err != nil || u.HomeDir == "" {
		return "", false
	}
	path := filepath.Join(u.HomeDir, ".config", "lazyfirewall", "config.toml")
	if path == primary {
		return "", false
	}
	return path, true
}

func stripComment(line string) string {
	inQuotes := false
	escaped := false

	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if inQuotes && r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if r == '#' && !inQuotes {
			return line[:i]
		}
	}

	return line
}

func parseString(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("empty string value")
	}
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) >= 2 {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid string %q", value)
		}
		return unquoted, nil
	}
	return "", fmt.Errorf("string must be quoted")
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool %q", value)
	}
}

func parseInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty number")
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", value)
	}
	if n < 0 {
		return 0, fmt.Errorf("number must be >= 0")
	}
	return n, nil
}

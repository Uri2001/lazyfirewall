package config

import (
	"os"
	"testing"
)

func TestParseString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "quoted value", input: `"debug"`, want: "debug", wantErr: false},
		{name: "escaped chars", input: `"val\"ue"`, want: `val"ue`, wantErr: false},
		{name: "unquoted", input: "debug", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseString(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("parseString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{input: "true", want: true, wantErr: false},
		{input: "FALSE", want: false, wantErr: false},
		{input: "yes", wantErr: true},
	}

	for _, tt := range tests {
		got, err := parseBool(tt.input)
		if (err != nil) != tt.wantErr {
			t.Fatalf("parseBool(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Fatalf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{input: "0", want: 0, wantErr: false},
		{input: "15", want: 15, wantErr: false},
		{input: "-1", wantErr: true},
		{input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		got, err := parseInt(tt.input)
		if (err != nil) != tt.wantErr {
			t.Fatalf("parseInt(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Fatalf("parseInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeConfig(t *testing.T) {
	cfg := Config{
		UI: UIConfig{
			Theme: "custom",
		},
		Behavior: BehaviorConfig{
			AutoRefreshSeconds: 5,
		},
	}

	warnings := normalizeConfig(&cfg)
	if cfg.UI.Theme != "default" {
		t.Fatalf("theme = %q, want default", cfg.UI.Theme)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected warnings, got none")
	}
}

func TestNormalizeConfig_NoWarningsForDefaults(t *testing.T) {
	cfg := Default()
	warnings := normalizeConfig(&cfg)
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for default config, got %v", warnings)
	}
}

func TestParse_ConfigSections(t *testing.T) {
	raw := `
[ui]
theme = "default"

[behavior]
default_permanent = true
auto_refresh_interval = 0

[advanced]
log_level = "debug"
`
	cfg := Default()
	warnings, err := parse(raw, &cfg)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if !cfg.Behavior.DefaultPermanent {
		t.Fatalf("default_permanent was not parsed")
	}
	if cfg.Advanced.LogLevel != "debug" {
		t.Fatalf("log_level = %q, want debug", cfg.Advanced.LogLevel)
	}
}

func TestParse_UnknownKeysProduceWarnings(t *testing.T) {
	raw := `
[ui]
unknown = "x"

[unknown_section]
foo = "bar"
`
	cfg := Default()
	warnings, err := parse(raw, &cfg)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if len(warnings) < 2 {
		t.Fatalf("expected warnings for unknown keys/section, got %v", warnings)
	}
}

func TestCandidatePaths_EnvOverride(t *testing.T) {
	old := os.Getenv("LAZYFIREWALL_CONFIG")
	if err := os.Setenv("LAZYFIREWALL_CONFIG", "/tmp/custom-config.toml"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("LAZYFIREWALL_CONFIG", old)
	})

	paths, err := candidatePaths()
	if err != nil {
		t.Fatalf("candidatePaths() error = %v", err)
	}
	if len(paths) != 1 || paths[0] != "/tmp/custom-config.toml" {
		t.Fatalf("candidatePaths() = %v, want [/tmp/custom-config.toml]", paths)
	}
}

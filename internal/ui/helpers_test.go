//go:build linux
// +build linux

package ui

import "testing"

func TestParsePortInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "slash format", input: "80/tcp", wantErr: false},
		{name: "space format", input: "53 udp", wantErr: false},
		{name: "bad port", input: "0/tcp", wantErr: true},
		{name: "bad protocol", input: "80/icmp", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePortInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePortInput(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseIPSetInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantType string
		wantErr  bool
	}{
		{name: "name only", input: "allowlist", wantName: "allowlist", wantType: "hash:ip", wantErr: false},
		{name: "name and type", input: "allowlist hash:net", wantName: "allowlist", wantType: "hash:net", wantErr: false},
		{name: "too many args", input: "a b c", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotType, err := parseIPSetInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseIPSetInput(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if err == nil {
				if gotName != tt.wantName || gotType != tt.wantType {
					t.Fatalf("parseIPSetInput(%q) = (%q, %q), want (%q, %q)", tt.input, gotName, gotType, tt.wantName, tt.wantType)
				}
			}
		})
	}
}

func TestMatchIndices(t *testing.T) {
	items := []string{"ssh", "http", "https", "dns"}
	got := matchIndices(items, "http")
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("matchIndices() = %v, want [1 2]", got)
	}
}

func TestLogMatchesSource(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{line: "firewalld[123]: changed zone", want: true},
		{line: "kernel: IN=eth0 OUT= DROP", want: true},
		{line: "unrelated daemon", want: false},
	}

	for _, tt := range tests {
		got := logMatchesSource(tt.line)
		if got != tt.want {
			t.Fatalf("logMatchesSource(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestLogMatchesZone(t *testing.T) {
	tests := []struct {
		name string
		line string
		zone string
		want bool
	}{
		{name: "zone filter disabled", line: "anything", zone: "", want: true},
		{name: "matches zone field", line: "firewalld: zone=public update", zone: "public", want: true},
		{name: "does not match zone field", line: "firewalld: zone=dmz update", zone: "public", want: false},
		{name: "line without zone key", line: "firewalld: reload done", zone: "public", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logMatchesZone(tt.line, tt.zone)
			if got != tt.want {
				t.Fatalf("logMatchesZone(%q, %q) = %v, want %v", tt.line, tt.zone, got, tt.want)
			}
		})
	}
}

func TestCommonPrefixAndLimitList(t *testing.T) {
	prefix := commonPrefix([]string{"alpha", "alpine", "alps"})
	if prefix != "alp" {
		t.Fatalf("commonPrefix() = %q, want %q", prefix, "alp")
	}

	l := limitList([]string{"a", "b", "c", "d"}, 2)
	if len(l) != 3 || l[0] != "a" || l[1] != "b" || l[2] != "…" {
		t.Fatalf("limitList() = %v, want [a b …]", l)
	}
}

func TestValidateRichRule(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: "rule service name=ssh accept", wantErr: false},
		{name: "empty", value: "   ", wantErr: true},
		{name: "missing prefix", value: "service name=ssh accept", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRichRule(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRichRule(%q) error = %v, wantErr = %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestIndexOfZone(t *testing.T) {
	zones := []string{"public", "home", "dmz"}
	if got := indexOfZone(zones, "home"); got != 1 {
		t.Fatalf("indexOfZone(home) = %d, want 1", got)
	}
	if got := indexOfZone(zones, "work"); got != -1 {
		t.Fatalf("indexOfZone(work) = %d, want -1", got)
	}
}

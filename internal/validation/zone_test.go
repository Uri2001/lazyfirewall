package validation

import (
	"strings"
	"testing"
)

func TestIsValidZoneName(t *testing.T) {
	tests := []struct {
		name    string
		zone    string
		wantErr bool
	}{
		{name: "valid simple", zone: "public", wantErr: false},
		{name: "valid dash", zone: "my-zone", wantErr: false},
		{name: "valid underscore", zone: "my_zone", wantErr: false},
		{name: "path traversal", zone: "../etc/passwd", wantErr: true},
		{name: "absolute path", zone: "/etc/passwd", wantErr: true},
		{name: "windows path", zone: "..\\..\\windows", wantErr: true},
		{name: "empty", zone: "", wantErr: true},
		{name: "too long", zone: strings.Repeat("a", 200), wantErr: true},
		{name: "invalid chars", zone: "zone@#$%", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidZoneName(tt.zone)
			if (err != nil) != tt.wantErr {
				t.Fatalf("IsValidZoneName(%q) error = %v, wantErr = %v", tt.zone, err, tt.wantErr)
			}
		})
	}
}

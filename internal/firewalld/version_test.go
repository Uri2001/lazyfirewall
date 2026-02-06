//go:build linux
// +build linux

package firewalld

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version string
		want    APIVersion
	}{
		{version: "0.3.0", want: APIv1},
		{version: "0.9.9", want: APIv1},
		{version: "1.0.0", want: APIv2},
		{version: "1.2.3", want: APIv2},
		{version: "2.0.0", want: APIv2},
		{version: "", want: APIUnknown},
		{version: "invalid", want: APIUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := parseVersion(tt.version)
			if got != tt.want {
				t.Fatalf("parseVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

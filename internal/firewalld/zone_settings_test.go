//go:build linux
// +build linux

package firewalld

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestParseZoneSettings(t *testing.T) {
	settings := map[string]dbus.Variant{
		"services":             dbus.MakeVariant([]string{"ssh", "http"}),
		"ports":                dbus.MakeVariant([][]string{{"22", "tcp"}, {"53", "udp"}}),
		"masquerade":           dbus.MakeVariant(true),
		"rules_str":            dbus.MakeVariant([]interface{}{"rule family=ipv4 accept"}),
		"interfaces":           dbus.MakeVariant([]string{"eth0"}),
		"sources":              dbus.MakeVariant([]string{"10.0.0.0/24"}),
		"target":               dbus.MakeVariant("DROP"),
		"icmp_blocks":          dbus.MakeVariant([]string{"echo-request"}),
		"icmp_block_inversion": dbus.MakeVariant(true),
		"short":                dbus.MakeVariant("Public"),
		"description":          dbus.MakeVariant("Test zone"),
	}

	z, err := parseZoneSettings("public", settings)
	if err != nil {
		t.Fatalf("parseZoneSettings() error = %v, want nil", err)
	}
	if z.Name != "public" {
		t.Fatalf("zone name = %q, want public", z.Name)
	}
	if len(z.Services) != 2 || z.Services[0] != "ssh" || z.Services[1] != "http" {
		t.Fatalf("services = %#v, want [ssh http]", z.Services)
	}
	if len(z.Ports) != 2 || z.Ports[0] != (Port{Port: "22", Protocol: "tcp"}) {
		t.Fatalf("ports = %#v, unexpected", z.Ports)
	}
	if !z.Masquerade {
		t.Fatalf("masquerade = false, want true")
	}
	if len(z.RichRules) != 1 || z.RichRules[0] != "rule family=ipv4 accept" {
		t.Fatalf("rich rules = %#v, unexpected", z.RichRules)
	}
	if len(z.Interfaces) != 1 || z.Interfaces[0] != "eth0" {
		t.Fatalf("interfaces = %#v, unexpected", z.Interfaces)
	}
	if len(z.Sources) != 1 || z.Sources[0] != "10.0.0.0/24" {
		t.Fatalf("sources = %#v, unexpected", z.Sources)
	}
	if z.Target != "DROP" {
		t.Fatalf("target = %q, want DROP", z.Target)
	}
	if len(z.IcmpBlocks) != 1 || z.IcmpBlocks[0] != "echo-request" {
		t.Fatalf("icmp blocks = %#v, unexpected", z.IcmpBlocks)
	}
	if !z.IcmpInvert {
		t.Fatalf("icmp invert = false, want true")
	}
	if z.Short != "Public" {
		t.Fatalf("short = %q, want Public", z.Short)
	}
	if z.Description != "Test zone" {
		t.Fatalf("description = %q, want Test zone", z.Description)
	}
}

func TestVariantToPorts(t *testing.T) {
	tests := []struct {
		name    string
		input   dbus.Variant
		want    []Port
		wantErr bool
	}{
		{
			name:  "tuple list",
			input: dbus.MakeVariant([][]string{{"80", "tcp"}}),
			want:  []Port{{Port: "80", Protocol: "tcp"}},
		},
		{
			name:  "string list",
			input: dbus.MakeVariant([]string{"53/udp"}),
			want:  []Port{{Port: "53", Protocol: "udp"}},
		},
		{
			name:  "interface tuples",
			input: dbus.MakeVariant([]interface{}{[]interface{}{"22", "tcp"}, []string{"443", "tcp"}}),
			want:  []Port{{Port: "22", Protocol: "tcp"}, {Port: "443", Protocol: "tcp"}},
		},
		{
			name:    "invalid input",
			input:   dbus.MakeVariant(map[string]string{"p": "80/tcp"}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := variantToPorts(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("variantToPorts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(ports) = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("port[%d] = %#v, want %#v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestVariantToStringSlice(t *testing.T) {
	got := variantToStringSlice(dbus.MakeVariant([]interface{}{"a", 1, "b"}))
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("variantToStringSlice() = %#v, want [a b]", got)
	}
}

func TestIsInvalidZone(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "dbus name",
			err:  &dbus.Error{Name: "org.fedoraproject.FirewallD1.Error.INVALID_ZONE"},
			want: true,
		},
		{
			name: "message text",
			err:  errors.New("invalid zone: does not exist"),
			want: true,
		},
		{
			name: "other",
			err:  errors.New("temporary failure"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInvalidZone(tt.err); got != tt.want {
				t.Fatalf("isInvalidZone(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

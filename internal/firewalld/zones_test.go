//go:build linux
// +build linux

package firewalld

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestNormalizeActiveZones(t *testing.T) {
	t.Run("map string slice", func(t *testing.T) {
		in := map[string][]string{"public": {"eth0"}}
		got, err := normalizeActiveZones(in)
		if err != nil {
			t.Fatalf("normalizeActiveZones() error = %v, want nil", err)
		}
		if len(got["public"]) != 1 || got["public"][0] != "eth0" {
			t.Fatalf("public refs = %#v, want [eth0]", got["public"])
		}
	})

	t.Run("nested map with dedupe", func(t *testing.T) {
		in := map[string]map[string][]string{
			"public": {
				"interfaces": {"eth0", "eth0"},
				"sources":    {"10.0.0.0/24"},
			},
		}
		got, err := normalizeActiveZones(in)
		if err != nil {
			t.Fatalf("normalizeActiveZones() error = %v, want nil", err)
		}
		if len(got["public"]) != 2 {
			t.Fatalf("public refs len = %d, want 2", len(got["public"]))
		}
	})

	t.Run("variant map", func(t *testing.T) {
		in := map[string]dbus.Variant{
			"public": dbus.MakeVariant([]interface{}{"eth0", "10.0.0.0/24", "eth0"}),
		}
		got, err := normalizeActiveZones(in)
		if err != nil {
			t.Fatalf("normalizeActiveZones() error = %v, want nil", err)
		}
		if len(got["public"]) != 2 {
			t.Fatalf("public refs len = %d, want 2", len(got["public"]))
		}
	})

	t.Run("wrapped variant", func(t *testing.T) {
		in := dbus.MakeVariant(map[string][]string{
			"public": {"eth0"},
		})
		got, err := normalizeActiveZones(in)
		if err != nil {
			t.Fatalf("normalizeActiveZones() error = %v, want nil", err)
		}
		if len(got["public"]) != 1 || got["public"][0] != "eth0" {
			t.Fatalf("public refs = %#v, want [eth0]", got["public"])
		}
	})

	t.Run("unexpected type", func(t *testing.T) {
		if _, err := normalizeActiveZones(42); err == nil {
			t.Fatalf("normalizeActiveZones() error = nil, want error")
		}
	})
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  []string
	}{
		{
			name:  "string slice",
			input: []string{"a", "b"},
			want:  []string{"a", "b"},
		},
		{
			name:  "interface slice",
			input: []interface{}{"a", 1, "b"},
			want:  []string{"a", "b"},
		},
		{
			name: "map interface",
			input: map[string]interface{}{
				"interfaces": []string{"eth0"},
				"sources":    []interface{}{"10.0.0.0/24"},
			},
			want: []string{"eth0", "10.0.0.0/24"},
		},
		{
			name:  "variant",
			input: dbus.MakeVariant([]string{"x"}),
			want:  []string{"x"},
		},
		{
			name:  "unsupported",
			input: 123,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStringSlice(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("len(toStringSlice()) = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("toStringSlice()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDedupeStrings(t *testing.T) {
	got := dedupeStrings([]string{"a", "b", "a", "c", "b"})
	if len(got) != 3 {
		t.Fatalf("len(dedupeStrings()) = %d, want 3", len(got))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("dedupeStrings() = %#v, want [a b c]", got)
	}
}

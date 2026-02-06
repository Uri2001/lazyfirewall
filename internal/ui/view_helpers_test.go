//go:build linux
// +build linux

package ui

import (
	"strings"
	"testing"

	"lazyfirewall/internal/firewalld"
)

func TestFilterMissingServices(t *testing.T) {
	template := []string{"ssh", "http", "dns"}
	current := []string{"ssh", "dns"}
	missing := filterMissingServices(template, current)
	if len(missing) != 1 || missing[0] != "http" {
		t.Fatalf("filterMissingServices() = %v, want [http]", missing)
	}
}

func TestModeLabel(t *testing.T) {
	if got := modeLabel(true); got != "permanent" {
		t.Fatalf("modeLabel(true) = %q, want permanent", got)
	}
	if got := modeLabel(false); got != "runtime" {
		t.Fatalf("modeLabel(false) = %q, want runtime", got)
	}
}

func TestFilterMissingPorts(t *testing.T) {
	template := []firewalld.Port{
		{Port: "22", Protocol: "tcp"},
		{Port: "53", Protocol: "udp"},
	}
	current := []firewalld.Port{
		{Port: "22", Protocol: "tcp"},
	}
	missing := filterMissingPorts(template, current)
	if len(missing) != 1 || missing[0].Port != "53" || missing[0].Protocol != "udp" {
		t.Fatalf("filterMissingPorts() = %+v, want 53/udp", missing)
	}
}

func TestAttachedIPSets(t *testing.T) {
	z := &firewalld.Zone{
		Sources: []string{"10.0.0.0/24", "ipset:blocklist", "ipset:allowlist"},
	}
	sets := attachedIPSets(z)
	if len(sets) != 2 {
		t.Fatalf("attachedIPSets() len = %d, want 2", len(sets))
	}
	if _, ok := sets["blocklist"]; !ok {
		t.Fatalf("missing blocklist in attachedIPSets")
	}
}

func TestFormatBytes(t *testing.T) {
	if got := formatBytes(512); got != "512 B" {
		t.Fatalf("formatBytes(512) = %q", got)
	}
	if got := formatBytes(2048); !strings.Contains(got, "KB") {
		t.Fatalf("formatBytes(2048) = %q, expected KB", got)
	}
}

func TestPortDiffMark(t *testing.T) {
	perm := &firewalld.Zone{
		Ports: []firewalld.Port{
			{Port: "80", Protocol: "tcp"},
			{Port: "53", Protocol: "udp"},
		},
	}

	if got := portDiffMark(firewalld.Port{Port: "80", Protocol: "tcp"}, perm); got != "" {
		t.Fatalf("portDiffMark exact match = %q, want empty", got)
	}
	if got := portDiffMark(firewalld.Port{Port: "53", Protocol: "tcp"}, perm); got != "~" {
		t.Fatalf("portDiffMark protocol change = %q, want ~", got)
	}
	if got := portDiffMark(firewalld.Port{Port: "443", Protocol: "tcp"}, perm); got != "*" {
		t.Fatalf("portDiffMark new port = %q, want *", got)
	}
}

func TestHighlightMatch(t *testing.T) {
	base := "allow-http-service"
	if got := highlightMatch(base, ""); got != base {
		t.Fatalf("highlightMatch empty query changed text: %q", got)
	}
	got := highlightMatch(base, "http")
	if !strings.Contains(got, "http") {
		t.Fatalf("highlightMatch output missing matched fragment: %q", got)
	}
}

func TestDiffFunctionsBasic(t *testing.T) {
	runtime := &firewalld.Zone{
		Services:   []string{"ssh", "http"},
		Ports:      []firewalld.Port{{Port: "22", Protocol: "tcp"}},
		RichRules:  []string{"rule service name=ssh accept"},
		Interfaces: []string{"eth0"},
		Sources:    []string{"10.0.0.0/24"},
		Masquerade: true,
		Target:     "ACCEPT",
		IcmpBlocks: []string{"echo-request"},
	}
	permanent := &firewalld.Zone{
		Services:   []string{"ssh"},
		Ports:      []firewalld.Port{{Port: "53", Protocol: "udp"}},
		RichRules:  []string{"rule service name=ssh accept"},
		Interfaces: []string{"eth1"},
		Sources:    []string{"192.168.1.0/24"},
		Masquerade: false,
		Target:     "DROP",
		IcmpBlocks: []string{},
	}

	if left, right := diffServices(runtime, permanent); len(left) == 0 || len(right) == 0 {
		t.Fatalf("diffServices returned empty output")
	}
	if left, right := diffPorts(runtime, permanent); len(left) == 0 || len(right) == 0 {
		t.Fatalf("diffPorts returned empty output")
	}
	if left, right := diffRichRules(runtime, permanent); len(left) == 0 || len(right) == 0 {
		t.Fatalf("diffRichRules returned empty output")
	}
	if left, right := diffNetwork(runtime, permanent); len(left) == 0 || len(right) == 0 {
		t.Fatalf("diffNetwork returned empty output")
	}
	if left, right := diffInfo(runtime, permanent); len(left) == 0 || len(right) == 0 {
		t.Fatalf("diffInfo returned empty output")
	}
}

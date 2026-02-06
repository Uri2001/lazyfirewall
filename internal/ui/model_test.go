//go:build linux
// +build linux

package ui

import (
	"testing"

	"lazyfirewall/internal/firewalld"
)

func TestNewModelDefaults(t *testing.T) {
	m := NewModel(&firewalld.Client{}, Options{DryRun: true, DefaultPermanent: true})

	if !m.loading {
		t.Fatalf("loading = false, want true")
	}
	if !m.permanent {
		t.Fatalf("permanent = false, want true")
	}
	if !m.dryRun {
		t.Fatalf("dryRun = false, want true")
	}
	if m.focus != focusZones {
		t.Fatalf("focus = %v, want focusZones", m.focus)
	}
	if m.tab != tabServices {
		t.Fatalf("tab = %v, want tabServices", m.tab)
	}
	if m.inputMode != inputNone {
		t.Fatalf("inputMode = %v, want inputNone", m.inputMode)
	}
	if m.backupDone == nil {
		t.Fatalf("backupDone = nil, want initialized map")
	}
	if m.logLinesStore == nil {
		t.Fatalf("logLinesStore = nil, want initialized store")
	}
}

func TestNetworkItems(t *testing.T) {
	m := Model{
		runtimeData: &firewalld.Zone{
			Interfaces: []string{"eth0", "wlan0"},
			Sources:    []string{"10.0.0.0/24"},
		},
	}

	items := m.networkItems()
	if len(items) != 3 {
		t.Fatalf("len(networkItems) = %d, want 3", len(items))
	}
	if items[0] != (networkItem{kind: "iface", value: "eth0"}) {
		t.Fatalf("items[0] = %#v, unexpected", items[0])
	}
	if items[2] != (networkItem{kind: "source", value: "10.0.0.0/24"}) {
		t.Fatalf("items[2] = %#v, unexpected", items[2])
	}
}

func TestEnsureLogLinesStoreInitializes(t *testing.T) {
	m := Model{}
	if m.logLinesStore != nil {
		t.Fatalf("logLinesStore precondition violated: expected nil")
	}

	m.appendLogLine("line")
	if m.logLinesStore == nil {
		t.Fatalf("logLinesStore = nil after append, want initialized")
	}

	lines := m.getLogLines()
	if len(lines) != 1 || lines[0] != "line" {
		t.Fatalf("getLogLines() = %#v, want [line]", lines)
	}
}

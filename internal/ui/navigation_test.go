//go:build linux
// +build linux

package ui

import (
	"testing"

	"lazyfirewall/internal/firewalld"
)

func TestTabsCycle(t *testing.T) {
	m := Model{tab: tabServices}
	m.nextTab()
	if m.tab != tabPorts {
		t.Fatalf("nextTab from services = %v, want %v", m.tab, tabPorts)
	}

	m.tab = tabServices
	m.prevTab()
	if m.tab != tabInfo {
		t.Fatalf("prevTab from services = %v, want %v", m.tab, tabInfo)
	}
}

func TestClampSelections(t *testing.T) {
	m := Model{
		runtimeData: &firewalld.Zone{
			Services:  []string{"ssh"},
			Ports:     []firewalld.Port{{Port: "80", Protocol: "tcp"}},
			RichRules: []string{"rule service name=ssh accept"},
		},
		serviceIndex: 10,
		portIndex:    10,
		richIndex:    10,
		networkIndex: 10,
		ipsets:       []string{"set1"},
		ipsetIndex:   10,
	}

	m.clampSelections()

	if m.serviceIndex != 0 || m.portIndex != 0 || m.richIndex != 0 || m.networkIndex != 0 || m.ipsetIndex != 0 {
		t.Fatalf("clampSelections did not clamp all indices: %+v", m)
	}
}

func TestMoveMainSelection_IPSets(t *testing.T) {
	m := Model{
		tab:        tabIPSets,
		ipsets:     []string{"a", "b", "c"},
		ipsetIndex: 1,
	}
	m.moveMainSelection(1)
	if m.ipsetIndex != 2 {
		t.Fatalf("ipsetIndex = %d, want 2", m.ipsetIndex)
	}
	m.moveMainSelection(1)
	if m.ipsetIndex != 2 {
		t.Fatalf("ipsetIndex should stay at upper bound, got %d", m.ipsetIndex)
	}
	m.moveMainSelection(-10)
	if m.ipsetIndex != 0 {
		t.Fatalf("ipsetIndex should clamp to 0, got %d", m.ipsetIndex)
	}
}

func TestCurrentItemsByTab(t *testing.T) {
	m := Model{
		runtimeData: &firewalld.Zone{
			Services:   []string{"ssh"},
			Ports:      []firewalld.Port{{Port: "53", Protocol: "udp"}},
			RichRules:  []string{"rule service name=ssh accept"},
			Interfaces: []string{"eth0"},
			Sources:    []string{"10.0.0.0/24"},
		},
		ipsets: []string{"set1"},
	}

	m.tab = tabServices
	if got := m.currentItems(); len(got) != 1 || got[0] != "ssh" {
		t.Fatalf("services currentItems = %v", got)
	}

	m.tab = tabPorts
	if got := m.currentItems(); len(got) != 1 || got[0] != "53/udp" {
		t.Fatalf("ports currentItems = %v", got)
	}

	m.tab = tabRich
	if got := m.currentItems(); len(got) != 1 {
		t.Fatalf("rich currentItems = %v", got)
	}

	m.tab = tabNetwork
	if got := m.currentItems(); len(got) != 2 {
		t.Fatalf("network currentItems = %v", got)
	}

	m.tab = tabIPSets
	if got := m.currentItems(); len(got) != 1 || got[0] != "set1" {
		t.Fatalf("ipsets currentItems = %v", got)
	}
}

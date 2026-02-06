//go:build linux
// +build linux

package ui

import (
	"testing"

	"lazyfirewall/internal/firewalld"
)

func TestCurrentData(t *testing.T) {
	r := &firewalld.Zone{Name: "runtime"}
	p := &firewalld.Zone{Name: "permanent"}

	m := Model{runtimeData: r, permanentData: p, permanent: false}
	if got := m.currentData(); got != r {
		t.Fatalf("currentData runtime = %v, want runtimeData", got)
	}

	m.permanent = true
	if got := m.currentData(); got != p {
		t.Fatalf("currentData permanent = %v, want permanentData", got)
	}
}

func TestCurrentIPSetName(t *testing.T) {
	m := Model{ipsets: []string{"set1", "set2"}, ipsetIndex: 1}
	if got := m.currentIPSetName(); got != "set2" {
		t.Fatalf("currentIPSetName() = %q, want set2", got)
	}

	m.ipsetIndex = 5
	if got := m.currentIPSetName(); got != "" {
		t.Fatalf("currentIPSetName() out of bounds = %q, want empty", got)
	}
}

func TestCurrentService(t *testing.T) {
	m := Model{
		runtimeData: &firewalld.Zone{
			Services: []string{"ssh", "http"},
		},
		serviceIndex: 1,
	}
	if got := m.currentService(); got != "http" {
		t.Fatalf("currentService() = %q, want http", got)
	}

	m.serviceIndex = 10
	if got := m.currentService(); got != "" {
		t.Fatalf("currentService() out of bounds = %q, want empty", got)
	}
}

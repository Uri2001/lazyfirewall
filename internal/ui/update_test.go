//go:build linux
// +build linux

package ui

import (
	"testing"

	"lazyfirewall/internal/firewalld"
)

func TestSubmitInput_ManualBackupAllowsEmptyDescription(t *testing.T) {
	m := NewModel(&firewalld.Client{}, Options{})
	m.zones = []string{"public"}
	m.selected = 0
	m.inputMode = inputManualBackup
	m.input.SetValue("")

	cmd := m.submitInput()
	if cmd == nil {
		t.Fatalf("submitInput() returned nil cmd for manual backup")
	}
	if m.err != nil {
		t.Fatalf("submitInput() error = %v, want nil", m.err)
	}
}

func TestLogLinesStoreSnapshotIsolation(t *testing.T) {
	m := NewModel(&firewalld.Client{}, Options{})
	m.appendLogLine("one")
	m.appendLogLine("two")

	lines := m.getLogLines()
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}
	lines[0] = "mutated"

	lines2 := m.getLogLines()
	if lines2[0] != "one" {
		t.Fatalf("log lines were mutated through snapshot")
	}
}

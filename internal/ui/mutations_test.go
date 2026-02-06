//go:build linux
// +build linux

package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMaybeBackup_Behavior(t *testing.T) {
	noop := func() tea.Msg { return nil }

	m := Model{}
	cmd := m.maybeBackup("public", false, noop)
	if cmd == nil {
		t.Fatalf("expected command when backup not required")
	}

	m = Model{readOnly: true}
	cmd = m.maybeBackup("public", true, noop)
	if cmd == nil {
		t.Fatalf("expected command in read-only mode")
	}

	m = Model{}
	cmd = m.maybeBackup("", true, noop)
	if cmd == nil {
		t.Fatalf("expected command for empty zone fallback")
	}

	m = Model{}
	cmd = m.maybeBackup("public", true, noop)
	if cmd == nil {
		t.Fatalf("expected backup command when backup required")
	}
	if m.pendingMutation == nil {
		t.Fatalf("pendingMutation should be set when backup is required")
	}

	m = Model{backupDone: map[string]bool{"public": true}}
	cmd = m.maybeBackup("public", true, noop)
	if cmd == nil {
		t.Fatalf("expected original command when backup already done")
	}
}

func TestSetDryRunNotice(t *testing.T) {
	m := Model{}
	m.setDryRunNotice("add service ssh")
	if m.err != nil {
		t.Fatalf("err should be cleared in dry-run notice")
	}
	if m.notice == "" {
		t.Fatalf("notice should be set")
	}
}

func TestPushUndoRedoLimits(t *testing.T) {
	m := Model{}

	for i := 0; i < undoLimit+5; i++ {
		m.pushUndo(undoAction{label: "u"}, false)
		m.pushRedo(undoAction{label: "r"})
	}

	if len(m.undoStack) != undoLimit {
		t.Fatalf("undo stack len = %d, want %d", len(m.undoStack), undoLimit)
	}
	if len(m.redoStack) != undoLimit {
		t.Fatalf("redo stack len = %d, want %d", len(m.redoStack), undoLimit)
	}
}

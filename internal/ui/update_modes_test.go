//go:build linux
// +build linux

package ui

import (
	"testing"

	"lazyfirewall/internal/backup"
	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleHelpMode(t *testing.T) {
	m := Model{helpMode: true}

	next, cmd, handled := m.handleHelpMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !handled {
		t.Fatalf("expected help mode to handle key")
	}
	if next.helpMode {
		t.Fatalf("help mode should be disabled after '?'")
	}
	if cmd != nil {
		t.Fatalf("expected nil command on help close")
	}
}

func TestHandleTemplateMode(t *testing.T) {
	m := Model{templateMode: true, templateIndex: 0}
	next, _, handled := m.handleTemplateMode(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Fatalf("expected template mode to handle down key")
	}
	if next.templateIndex <= 0 {
		t.Fatalf("template index should move down, got %d", next.templateIndex)
	}
}

func TestHandleBackupModeReadOnly(t *testing.T) {
	m := Model{
		backupMode:  true,
		readOnly:    true,
		backupItems: []backup.Backup{{Zone: "public"}},
	}
	next, cmd, handled := m.handleBackupMode(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected backup mode to handle enter")
	}
	if cmd != nil {
		t.Fatalf("expected nil command in read-only backup restore")
	}
	if next.err == nil {
		t.Fatalf("expected permission error in read-only backup restore")
	}
}

func TestHandleInputModeEscSearch(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("abc")

	m := Model{
		inputMode:    inputSearch,
		searchQuery:  "abc",
		input:        ti,
		detailsMode:  true,
		templateMode: true,
	}
	next, cmd, handled := m.handleInputMode(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Fatalf("expected input mode to handle esc")
	}
	if cmd != nil {
		t.Fatalf("expected nil command on esc")
	}
	if next.inputMode != inputNone {
		t.Fatalf("input mode should be cleared on esc")
	}
	if next.searchQuery != "" {
		t.Fatalf("search query should be reset on esc")
	}
}

func TestHandleDetailsMode(t *testing.T) {
	m := Model{detailsMode: true, detailsLoading: true, detailsErr: firewalld.ErrPermissionDenied}
	next, cmd, handled := m.handleDetailsMode(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatalf("expected details mode to handle enter")
	}
	if cmd != nil {
		t.Fatalf("expected nil command when closing details mode")
	}
	if next.detailsMode || next.detailsLoading || next.detailsErr != nil {
		t.Fatalf("details mode should be fully reset on close")
	}
}

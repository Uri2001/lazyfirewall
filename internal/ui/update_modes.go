//go:build linux
// +build linux

package ui

import (
	"fmt"

	"lazyfirewall/internal/firewalld"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleHelpMode(msg tea.Msg) (Model, tea.Cmd, bool) {
	if !m.helpMode {
		return m, nil, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}

	switch key.String() {
	case "esc", "?":
		m.helpMode = false
		return m, nil, true
	case "ctrl+c", "q":
		return m, tea.Quit, true
	default:
		return m, nil, true
	}
}

func (m Model) handleTemplateMode(msg tea.Msg) (Model, tea.Cmd, bool) {
	if !m.templateMode {
		return m, nil, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}

	switch key.String() {
	case "esc", "q", "t":
		m.templateMode = false
		return m, nil, true
	case "j", "down":
		if m.templateIndex < len(defaultTemplates)-1 {
			m.templateIndex++
		}
		return m, nil, true
	case "k", "up":
		if m.templateIndex > 0 {
			m.templateIndex--
		}
		return m, nil, true
	case "enter":
		return m, m.applyTemplate(), true
	default:
		return m, nil, false
	}
}

func (m Model) handleBackupMode(msg tea.Msg) (Model, tea.Cmd, bool) {
	if !m.backupMode {
		return m, nil, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}

	switch key.String() {
	case "esc", "ctrl+r":
		m.backupMode = false
		m.backupItems = nil
		m.backupPreview = ""
		m.backupErr = nil
		return m, nil, true
	case "j", "down":
		if len(m.backupItems) > 0 && m.backupIndex < len(m.backupItems)-1 {
			m.backupIndex++
			item := m.backupItems[m.backupIndex]
			return m, previewBackupCmd(item.Zone, item.Path, m.permanentData), true
		}
		return m, nil, true
	case "k", "up":
		if len(m.backupItems) > 0 && m.backupIndex > 0 {
			m.backupIndex--
			item := m.backupItems[m.backupIndex]
			return m, previewBackupCmd(item.Zone, item.Path, m.permanentData), true
		}
		return m, nil, true
	case "enter":
		if m.readOnly {
			m.err = firewalld.ErrPermissionDenied
			return m, nil, true
		}
		if len(m.backupItems) == 0 || m.backupIndex >= len(m.backupItems) {
			return m, nil, true
		}
		item := m.backupItems[m.backupIndex]
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("restore backup for zone %s", item.Zone))
			return m, nil, true
		}
		m.err = nil
		m.loading = true
		m.pendingZone = item.Zone
		return m, restoreBackupCmd(m.client, item.Zone, item), true
	default:
		return m, nil, true
	}
}

func (m Model) handleInputMode(msg tea.Msg) (Model, tea.Cmd, bool) {
	if m.inputMode == inputNone {
		return m, nil, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}

	if key.String() == "tab" {
		switch m.inputMode {
		case inputExportZone, inputImportZone:
			m.completePath()
			return m, nil, true
		case inputAddService:
			m.completeServiceName()
			return m, nil, true
		}
	}

	switch key.String() {
	case "ctrl+c":
		return m, tea.Quit, true
	case "esc":
		if m.inputMode == inputSearch {
			m.searchQuery = ""
			m.input.SetValue("")
		}
		if m.inputMode == inputEditRich {
			m.editRichOld = ""
		}
		if m.inputMode == inputPanicConfirm {
			m.panicCountdown = 0
		}
		m.inputMode = inputNone
		m.input.Blur()
		return m, nil, true
	case "enter":
		if m.inputMode == inputSearch {
			m.inputMode = inputNone
			m.input.Blur()
			return m, nil, true
		}
		return m, m.submitInput(), true
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	if m.inputMode == inputExportZone || m.inputMode == inputImportZone {
		m.notice = ""
	}
	if m.inputMode == inputSearch {
		m.searchQuery = m.input.Value()
		m.applySearchSelection()
		if m.tab == tabIPSets {
			if entriesCmd := m.fetchCurrentIPSetEntries(); entriesCmd != nil {
				return m, tea.Batch(cmd, entriesCmd), true
			}
		}
	}
	return m, cmd, true
}

func (m Model) handleDetailsMode(msg tea.Msg) (Model, tea.Cmd, bool) {
	if !m.detailsMode {
		return m, nil, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil, false
	}

	switch key.String() {
	case "esc", "enter":
		m.detailsMode = false
		m.detailsLoading = false
		m.detailsErr = nil
		return m, nil, true
	default:
		return m, nil, false
	}
}

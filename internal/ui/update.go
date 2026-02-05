//go:build linux
// +build linux

package ui

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"lazyfirewall/internal/backup"
	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchZonesCmd(m.client),
		fetchDefaultZoneCmd(m.client),
		fetchActiveZonesCmd(m.client),
		fetchPanicModeCmd(m.client),
		fetchIPSetsCmd(m.client, m.permanent),
		fetchServiceCatalogCmd(m.client),
		subscribeSignalsCmd(m.client),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.helpMode {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc", "?":
				m.helpMode = false
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			default:
				return m, nil
			}
		}
	}

	if m.templateMode {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc", "q", "t":
				m.templateMode = false
				return m, nil
			case "j", "down":
				if m.templateIndex < len(defaultTemplates)-1 {
					m.templateIndex++
				}
				return m, nil
			case "k", "up":
				if m.templateIndex > 0 {
					m.templateIndex--
				}
				return m, nil
			case "enter":
				return m, m.applyTemplate()
			}
		}
	}

	if m.backupMode {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc", "ctrl+r":
				m.backupMode = false
				m.backupItems = nil
				m.backupPreview = ""
				m.backupErr = nil
				return m, nil
			case "j", "down":
				if len(m.backupItems) > 0 && m.backupIndex < len(m.backupItems)-1 {
					m.backupIndex++
					item := m.backupItems[m.backupIndex]
					return m, previewBackupCmd(item.Zone, item.Path, m.permanentData)
				}
				return m, nil
			case "k", "up":
				if len(m.backupItems) > 0 && m.backupIndex > 0 {
					m.backupIndex--
					item := m.backupItems[m.backupIndex]
					return m, previewBackupCmd(item.Zone, item.Path, m.permanentData)
				}
				return m, nil
			case "enter":
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				if len(m.backupItems) == 0 || m.backupIndex >= len(m.backupItems) {
					return m, nil
				}
				item := m.backupItems[m.backupIndex]
				if m.dryRun {
					m.setDryRunNotice(fmt.Sprintf("restore backup for zone %s", item.Zone))
					return m, nil
				}
				m.err = nil
				m.loading = true
				m.pendingZone = item.Zone
				return m, restoreBackupCmd(m.client, item.Zone, item)
			}
			return m, nil
		}
	}

	if m.inputMode != inputNone {
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.String() == "tab" {
				switch m.inputMode {
				case inputExportZone, inputImportZone:
					m.completePath()
					return m, nil
				case inputAddService:
					m.completeServiceName()
					return m, nil
				}
			}
			switch key.String() {
			case "ctrl+c":
				return m, tea.Quit
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
				return m, nil
			case "enter":
				if m.inputMode == inputSearch {
					m.inputMode = inputNone
					m.input.Blur()
					return m, nil
				}
				return m, m.submitInput()
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
						return m, tea.Batch(cmd, entriesCmd)
					}
				}
			}
			return m, cmd
		}
	}

	if m.detailsMode {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc", "enter":
				m.detailsMode = false
				m.detailsLoading = false
				m.detailsErr = nil
				return m, nil
			}
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.focus == focusZones {
				m.focus = focusMain
			} else {
				m.focus = focusZones
			}
			return m, nil
		case "1":
			m.detailsMode = false
			m.tab = tabServices
			return m, nil
		case "2":
			m.detailsMode = false
			m.tab = tabPorts
			return m, nil
		case "3":
			m.detailsMode = false
			m.tab = tabRich
			return m, nil
		case "4":
			m.detailsMode = false
			m.tab = tabNetwork
			return m, nil
		case "5":
			m.detailsMode = false
			m.tab = tabIPSets
			if m.focus == focusMain {
				return m, m.fetchCurrentIPSetEntries()
			}
			return m, nil
		case "6":
			m.detailsMode = false
			m.tab = tabInfo
			return m, nil
		case "h", "left":
			m.detailsMode = false
			m.prevTab()
			if m.tab == tabIPSets {
				return m, m.fetchCurrentIPSetEntries()
			}
			return m, nil
		case "l", "right":
			m.detailsMode = false
			m.nextTab()
			if m.tab == tabIPSets {
				return m, m.fetchCurrentIPSetEntries()
			}
			return m, nil
		case "S":
			if m.tab == tabIPSets {
				m.err = fmt.Errorf("split view not available for IPSets")
				return m, nil
			}
			m.splitView = !m.splitView
			return m, nil
		case "?":
			m.helpMode = !m.helpMode
			if m.helpMode {
				m.templateMode = false
				m.detailsMode = false
				m.inputMode = inputNone
				m.input.Blur()
			}
			return m, nil
		case "t":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			m.templateMode = true
			m.templateIndex = 0
			return m, nil
		case "ctrl+e":
			if len(m.zones) == 0 || m.selected >= len(m.zones) {
				return m, nil
			}
			current := m.currentData()
			if current == nil {
				m.err = fmt.Errorf("no data loaded")
				return m, nil
			}
			m.err = nil
			m.notice = ""
			zone := m.zones[m.selected]
			m.inputMode = inputExportZone
			m.input.Placeholder = "export path (.json or .xml)"
			m.input.SetValue(defaultExportPath(zone))
			m.input.CursorEnd()
			m.input.Focus()
			return m, nil
		case "alt+i", "alt+I":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.zones) == 0 || m.selected >= len(m.zones) {
				return m, nil
			}
			m.err = nil
			m.notice = ""
			m.inputMode = inputImportZone
			m.input.Placeholder = "import path (.json or .xml)"
			m.input.SetValue("")
			m.input.CursorEnd()
			m.input.Focus()
			return m, nil
		case "ctrl+r":
			if len(m.zones) == 0 || m.selected >= len(m.zones) {
				return m, nil
			}
			m.err = nil
			m.notice = ""
			m.backupMode = true
			m.backupIndex = 0
			m.backupPreview = ""
			m.backupErr = nil
			zone := m.zones[m.selected]
			return m, fetchBackupsCmd(zone)
		case "alt+p", "alt+P":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			m.err = nil
			if m.dryRun {
				if m.panicMode {
					m.setDryRunNotice("disable panic mode")
				} else {
					m.setDryRunNotice("enable panic mode")
				}
				return m, nil
			}
			if m.panicMode {
				m.panicAutoArmed = false
				return m, disablePanicModeCmd(m.client)
			}
			m.inputMode = inputPanicConfirm
			m.input.SetValue("")
			m.input.Placeholder = "type YES to confirm"
			m.input.CursorEnd()
			m.input.Focus()
			m.panicCountdown = 5
			return m, panicTickCmd()
		case "/":
			if m.splitView {
				m.err = fmt.Errorf("search disabled in split view")
				return m, nil
			}
			m.err = nil
			m.inputMode = inputSearch
			m.input.Placeholder = "search"
			m.input.SetValue(m.searchQuery)
			m.input.CursorEnd()
			m.input.Focus()
			return m, nil
		case "n":
			if m.focus == focusZones {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.startAddZone()
			}
			if m.focus == focusMain && m.tab == tabIPSets && m.searchQuery == "" {
				return m, m.startAddIPSet()
			}
			if m.searchQuery != "" && !m.splitView && m.focus == focusMain {
				m.moveMatchSelection(true)
				if m.tab == tabIPSets {
					return m, m.fetchCurrentIPSetEntries()
				}
			}
			return m, nil
		case "N":
			if m.searchQuery != "" && !m.splitView && m.focus == focusMain {
				m.moveMatchSelection(false)
				if m.tab == tabIPSets {
					return m, m.fetchCurrentIPSetEntries()
				}
			}
			return m, nil
		case "r":
			m.loading = true
			m.err = nil
			m.notice = ""
			m.runtimeDenied = false
			m.permanentDenied = false
			m.runtimeInvalid = false
			m.runtimeData = nil
			m.permanentData = nil
			m.editRichOld = ""
			m.ipsetLoading = true
			return m, tea.Batch(fetchZonesCmd(m.client), fetchDefaultZoneCmd(m.client), fetchActiveZonesCmd(m.client), fetchPanicModeCmd(m.client), fetchIPSetsCmd(m.client, m.permanent))
		case "ctrl+b":
			return m, m.startManualBackup()
		case "c":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.zones) == 0 {
				return m, nil
			}
			zone := m.zones[m.selected]
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("commit runtime → permanent for zone %s", zone))
				return m, nil
			}
			m.loading = true
			m.err = nil
			m.notice = ""
			m.pendingZone = zone
			return m, m.maybeBackup(zone, true, commitRuntimeCmd(m.client, zone, nil, recordNone, false))
		case "u":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.zones) == 0 {
				return m, nil
			}
			zone := m.zones[m.selected]
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("reload firewalld for zone %s (revert runtime)", zone))
				return m, nil
			}
			m.loading = true
			m.err = nil
			m.notice = ""
			m.pendingZone = zone
			return m, reloadCmd(m.client, zone, nil, recordNone, false)
		case "ctrl+z":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.undoStack) == 0 {
				m.notice = "Nothing to undo"
				return m, nil
			}
			action := m.undoStack[len(m.undoStack)-1]
			m.undoStack = m.undoStack[:len(m.undoStack)-1]
			m.loading = true
			m.err = nil
			m.notice = ""
			m.pendingZone = action.zone
			return m, action.undo
		case "ctrl+y":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.redoStack) == 0 {
				m.notice = "Nothing to redo"
				return m, nil
			}
			action := m.redoStack[len(m.redoStack)-1]
			m.redoStack = m.redoStack[:len(m.redoStack)-1]
			m.loading = true
			m.err = nil
			m.notice = ""
			m.pendingZone = action.zone
			return m, action.redo
		case "P":
			m.permanent = !m.permanent
			if len(m.zones) > 0 && m.selected < len(m.zones) {
				m.err = nil
				return m, m.startZoneLoad(m.zones[m.selected], false)
			}
			m.ipsetLoading = true
			return m, fetchIPSetsCmd(m.client, m.permanent)
		case "j", "down":
			if m.focus == focusZones {
				if len(m.zones) > 0 && m.selected < len(m.zones)-1 {
					m.selected++
					m.err = nil
					return m, m.startZoneLoad(m.zones[m.selected], true)
				}
				return m, nil
			}
			m.moveMainSelection(1)
			if m.tab == tabIPSets {
				return m, m.fetchCurrentIPSetEntries()
			}
			return m, nil
		case "k", "up":
			if m.focus == focusZones {
				if len(m.zones) > 0 && m.selected > 0 {
					m.selected--
					m.err = nil
					return m, m.startZoneLoad(m.zones[m.selected], true)
				}
				return m, nil
			}
			m.moveMainSelection(-1)
			if m.tab == tabIPSets {
				return m, m.fetchCurrentIPSetEntries()
			}
			return m, nil
		case "enter":
			if m.focus == focusMain && m.tab == tabServices {
				service := m.currentService()
				if service == "" {
					return m, nil
				}
				if m.detailsMode && m.detailsName == service {
					m.detailsMode = false
					return m, nil
				}
				m.detailsMode = true
				m.detailsLoading = true
				m.detailsErr = nil
				m.detailsName = service
				return m, fetchServiceDetailsCmd(m.client, service)
			}
			return m, nil
		case "a":
			if m.focus == focusMain {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.startAddInput()
			}
			return m, nil
		case "i":
			if m.focus == focusMain && m.tab == tabNetwork {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.startAddInterface()
			}
			return m, nil
		case "s":
			if m.focus == focusMain && m.tab == tabNetwork {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.startAddSource()
			}
			return m, nil
		case "m":
			if m.focus == focusMain && m.tab == tabNetwork {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.toggleMasquerade()
			}
			return m, nil
		case "e":
			if m.focus == focusMain && m.tab == tabRich {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.startEditRich()
			}
			return m, nil
		case "d":
			if m.focus == focusZones {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.startDeleteZone()
			}
			if m.focus == focusMain {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				return m, m.removeSelected()
			}
			return m, nil
		case "D":
			if m.focus == focusZones {
				if m.readOnly {
					m.err = firewalld.ErrPermissionDenied
					return m, nil
				}
				if len(m.zones) == 0 || m.selected >= len(m.zones) {
					return m, nil
				}
				zone := m.zones[m.selected]
				if m.dryRun {
					m.setDryRunNotice(fmt.Sprintf("set default zone to %s", zone))
					return m, nil
				}
				m.err = nil
				return m, setDefaultZoneCmd(m.client, zone)
			}
			if m.focus == focusMain && m.tab == tabIPSets {
				return m, m.startDeleteIPSet()
			}
			return m, nil
		}
	case zonesMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.zones = msg.zones
		if len(m.zones) == 0 {
			m.err = fmt.Errorf("no zones returned")
			return m, nil
		}
		if m.pendingZone != "" {
			if idx := indexOfZone(m.zones, m.pendingZone); idx >= 0 {
				m.selected = idx
			} else if m.selected >= len(m.zones) {
				m.selected = 0
			}
		} else if m.selected >= len(m.zones) {
			m.selected = 0
		}
		cmd := m.startZoneLoad(m.zones[m.selected], !m.signalRefresh)
		m.signalRefresh = false
		cmds := []tea.Cmd{fetchDefaultZoneCmd(m.client), fetchActiveZonesCmd(m.client)}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	case activeZonesMsg:
		if msg.err != nil {
			if errors.Is(msg.err, firewalld.ErrPermissionDenied) || errors.Is(msg.err, firewalld.ErrUnsupportedAPI) {
				return m, nil
			}
			m.err = msg.err
			return m, nil
		}
		m.activeZones = msg.zones
		return m, nil
	case signalsReadyMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.signals = msg.ch
		m.signalsCancel = msg.cancel
		return m, listenSignalsCmd(m.signals)
	case signalsClosedMsg:
		m.signals = nil
		m.signalsCancel = nil
		return m, nil
	case firewalldSignalMsg:
		if m.loading && m.signalRefresh {
			return m, listenSignalsCmd(m.signals)
		}
		if strings.HasSuffix(msg.event.Name, ".PanicModeEnabled") {
			m.panicMode = true
		} else if strings.HasSuffix(msg.event.Name, ".PanicModeDisabled") {
			m.panicMode = false
			m.panicAutoArmed = false
		}
		m.loading = true
		m.err = nil
		m.signalRefresh = true
		if len(m.zones) > 0 && m.selected < len(m.zones) {
			m.pendingZone = m.zones[m.selected]
		}
		return m, tea.Batch(
			fetchZonesCmd(m.client),
			fetchDefaultZoneCmd(m.client),
			fetchActiveZonesCmd(m.client),
			fetchPanicModeCmd(m.client),
			listenSignalsCmd(m.signals),
		)
	case panicModeMsg:
		if msg.err != nil {
			if errors.Is(msg.err, firewalld.ErrPermissionDenied) || errors.Is(msg.err, firewalld.ErrUnsupportedAPI) {
				return m, nil
			}
			m.err = msg.err
			return m, nil
		}
		m.panicMode = msg.enabled
		return m, nil
	case panicToggleMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.panicMode = msg.enabled
		if msg.enabled && m.panicAutoArmed {
			m.panicAutoArmed = false
			return m, panicAutoDisableCmd(m.panicAutoDur)
		}
		m.panicAutoArmed = false
		return m, nil
	case panicTickMsg:
		if m.inputMode != inputPanicConfirm {
			return m, nil
		}
		if m.panicCountdown > 0 {
			m.panicCountdown--
		}
		if m.panicCountdown > 0 {
			return m, panicTickCmd()
		}
		m.err = nil
		return m, nil
	case panicAutoDisableMsg:
		if !m.panicMode {
			return m, nil
		}
		if m.readOnly {
			return m, nil
		}
		return m, disablePanicModeCmd(m.client)
	case backupCreatedMsg:
		if msg.err != nil {
			if errors.Is(msg.err, os.ErrNotExist) {
				m.err = fmt.Errorf("backup skipped: zone XML not found")
				if m.pendingMutation != nil {
					cmd := m.pendingMutation
					m.pendingMutation = nil
					return m, cmd
				}
				return m, nil
			}
			m.err = msg.err
			m.pendingMutation = nil
			return m, nil
		}
		if m.backupDone == nil {
			m.backupDone = make(map[string]bool)
		}
		m.backupDone[msg.zone] = true
		if m.pendingMutation != nil {
			cmd := m.pendingMutation
			m.pendingMutation = nil
			return m, cmd
		}
		return m, nil
	case backupManualCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.notice = "Backup created"
		if msg.backup.Description != "" {
			m.notice = fmt.Sprintf("Backup created: %s", msg.backup.Description)
		}
		if m.backupMode {
			return m, fetchBackupsCmd(msg.zone)
		}
		return m, nil
	case backupsMsg:
		if msg.err != nil {
			m.backupErr = msg.err
			m.backupItems = nil
			m.backupPreview = ""
			return m, nil
		}
		m.backupItems = msg.items
		m.backupErr = nil
		if len(m.backupItems) > 0 {
			m.backupIndex = 0
			item := m.backupItems[m.backupIndex]
			return m, previewBackupCmd(item.Zone, item.Path, m.permanentData)
		}
		m.backupPreview = ""
		return m, nil
	case backupPreviewMsg:
		if msg.err != nil {
			m.backupPreview = ""
			m.backupErr = msg.err
			return m, nil
		}
		m.backupPreview = msg.preview
		return m, nil
	case backupRestoreMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.backupMode = false
		m.backupItems = nil
		m.backupPreview = ""
		m.backupErr = nil
		m.loading = true
		m.pendingZone = msg.zone
		return m, tea.Batch(
			fetchZoneSettingsCmd(m.client, msg.zone, false),
			fetchZoneSettingsCmd(m.client, msg.zone, true),
			fetchActiveZonesCmd(m.client),
		)
	case exportMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.notice = fmt.Sprintf("Exported to %s", msg.path)
		return m, nil
	case importMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.notice = fmt.Sprintf("Imported from file")
		m.loading = true
		m.pendingZone = msg.zone
		return m, tea.Batch(
			fetchZoneSettingsCmd(m.client, msg.zone, false),
			fetchZoneSettingsCmd(m.client, msg.zone, true),
			fetchActiveZonesCmd(m.client),
		)
	case defaultZoneMsg:
		if msg.err != nil {
			if errors.Is(msg.err, firewalld.ErrPermissionDenied) {
				return m, nil
			}
			m.err = msg.err
			return m, nil
		}
		m.defaultZone = msg.zone
		return m, nil
	case zoneSettingsMsg:
		if msg.zoneName != "" && msg.zoneName != m.pendingZone {
			return m, nil
		}
		if msg.err != nil {
			if errors.Is(msg.err, firewalld.ErrPermissionDenied) {
				if msg.permanent {
					m.permanentDenied = true
					m.permanentData = nil
				} else {
					m.runtimeDenied = true
					m.runtimeData = nil
				}
				if msg.permanent == m.permanent {
					m.loading = false
				}
				return m, nil
			}
			if errors.Is(msg.err, firewalld.ErrInvalidZone) && !msg.permanent {
				m.runtimeInvalid = true
				m.runtimeDenied = false
				m.runtimeData = nil
				if msg.permanent == m.permanent {
					m.loading = false
				}
				return m, nil
			}
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if msg.permanent {
			m.permanentData = msg.zone
			m.permanentDenied = false
		} else {
			m.runtimeData = msg.zone
			m.runtimeDenied = false
			m.runtimeInvalid = false
		}
		if msg.permanent == m.permanent {
			m.loading = false
		}
		m.clampSelections()
		return m, nil
	case ipsetsMsg:
		if msg.permanent != m.permanent {
			return m, nil
		}
		m.ipsetLoading = false
		if msg.err != nil {
			m.ipsetErr = msg.err
			m.ipsetDenied = errors.Is(msg.err, firewalld.ErrPermissionDenied)
			return m, nil
		}
		m.ipsetErr = nil
		m.ipsetDenied = false
		m.ipsets = msg.sets
		if len(m.ipsets) == 0 {
			m.ipsetIndex = 0
			m.ipsetEntries = nil
			m.ipsetEntryName = ""
			m.ipsetEntriesErr = nil
			m.ipsetEntriesLoading = false
			return m, nil
		}
		if m.ipsetIndex < 0 || m.ipsetIndex >= len(m.ipsets) {
			m.ipsetIndex = 0
		}
		return m, m.fetchCurrentIPSetEntries()
	case ipsetEntriesMsg:
		if msg.permanent != m.permanent {
			return m, nil
		}
		if msg.name != m.ipsetEntryName {
			return m, nil
		}
		m.ipsetEntriesLoading = false
		if msg.err != nil {
			m.ipsetEntriesErr = msg.err
			return m, nil
		}
		m.ipsetEntriesErr = nil
		m.ipsetEntries = msg.entries
		return m, nil
	case ipsetMutationMsg:
		if msg.err != nil {
			m.ipsetErr = msg.err
			return m, nil
		}
		m.ipsetErr = nil
		m.ipsetEntriesErr = nil
		m.ipsetLoading = true
		return m, fetchIPSetsCmd(m.client, m.permanent)
	case mutationMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if msg.action != nil {
			switch msg.record {
			case recordUndo:
				m.pushUndo(*msg.action, msg.clearRedo)
			case recordRedo:
				m.pushRedo(*msg.action)
			}
		}
		m.loading = true
		m.err = nil
		m.notice = ""
		m.pendingZone = msg.zone
		m.detailsMode = false
		m.templateMode = false
		m.detailsName = ""
		m.details = nil
		m.detailsErr = nil
		m.detailsLoading = false
		m.runtimeDenied = false
		m.permanentDenied = false
		m.runtimeInvalid = false
		m.runtimeData = nil
		m.permanentData = nil
		m.editRichOld = ""
		return m, tea.Batch(
			fetchZoneSettingsCmd(m.client, msg.zone, false),
			fetchZoneSettingsCmd(m.client, msg.zone, true),
			fetchActiveZonesCmd(m.client),
		)
	case serviceDetailsMsg:
		if msg.service != m.detailsName {
			return m, nil
		}
		m.detailsLoading = false
		if msg.err != nil {
			m.detailsErr = msg.err
			return m, nil
		}
		m.detailsErr = nil
		m.details = msg.info
		return m, nil
	case serviceCatalogMsg:
		m.servicesLoading = false
		if msg.err != nil {
			m.servicesErr = msg.err
			return m, nil
		}
		m.servicesErr = nil
		m.availableServices = msg.services
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) clampSelections() {
	current := m.currentData()
	if current == nil {
		return
	}
	if m.serviceIndex >= len(current.Services) {
		m.serviceIndex = 0
	}
	if m.portIndex >= len(current.Ports) {
		m.portIndex = 0
	}
	if m.richIndex >= len(current.RichRules) {
		m.richIndex = 0
	}
	items := m.networkItems()
	if len(items) == 0 {
		m.networkIndex = 0
	} else if m.networkIndex >= len(items) {
		m.networkIndex = 0
	}
	if len(m.ipsets) == 0 {
		m.ipsetIndex = 0
	} else if m.ipsetIndex >= len(m.ipsets) {
		m.ipsetIndex = 0
	}
}

func (m *Model) startZoneLoad(zone string, reset bool) tea.Cmd {
	m.loading = true
	m.pendingZone = zone
	m.ipsetLoading = true
	if reset {
		m.detailsMode = false
		m.templateMode = false
		m.detailsName = ""
		m.details = nil
		m.detailsErr = nil
		m.detailsLoading = false
		m.runtimeDenied = false
		m.permanentDenied = false
		m.runtimeInvalid = false
		m.editRichOld = ""
		m.runtimeData = nil
		m.permanentData = nil
	}

	return tea.Batch(
		fetchZoneSettingsCmd(m.client, zone, false),
		fetchZoneSettingsCmd(m.client, zone, true),
		fetchIPSetsCmd(m.client, m.permanent),
	)
}

func (m *Model) maybeBackup(zone string, needsBackup bool, cmd tea.Cmd) tea.Cmd {
	if !needsBackup {
		return cmd
	}
	if m.readOnly {
		return cmd
	}
	if zone == "" {
		return cmd
	}
	if m.backupDone == nil {
		m.backupDone = make(map[string]bool)
	}
	if m.backupDone[zone] {
		return cmd
	}
	m.pendingMutation = cmd
	return createBackupCmd(zone)
}

func (m *Model) setDryRunNotice(action string) {
	m.err = nil
	m.notice = "DRY RUN: would " + action
}

func modeLabel(permanent bool) string {
	if permanent {
		return "permanent"
	}
	return "runtime"
}

const undoLimit = 20

func (m *Model) pushUndo(action undoAction, clearRedo bool) {
	if len(m.undoStack) >= undoLimit {
		m.undoStack = m.undoStack[1:]
	}
	m.undoStack = append(m.undoStack, action)
	if clearRedo {
		m.redoStack = nil
	}
}

func (m *Model) pushRedo(action undoAction) {
	if len(m.redoStack) >= undoLimit {
		m.redoStack = m.redoStack[1:]
	}
	m.redoStack = append(m.redoStack, action)
}

func (m *Model) actionAddService(zone, service string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add service " + service, zone: zone}
	action.undo = removeServiceCmd(m.client, zone, service, permanent, action, recordRedo, false)
	action.redo = addServiceCmd(m.client, zone, service, permanent, action, recordUndo, false)
	return addServiceCmd(m.client, zone, service, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveService(zone, service string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove service " + service, zone: zone}
	action.undo = addServiceCmd(m.client, zone, service, permanent, action, recordRedo, false)
	action.redo = removeServiceCmd(m.client, zone, service, permanent, action, recordUndo, false)
	return removeServiceCmd(m.client, zone, service, permanent, action, recordUndo, true)
}

func (m *Model) actionAddPort(zone string, port firewalld.Port, permanent bool) tea.Cmd {
	label := port.Port + "/" + port.Protocol
	action := &undoAction{label: "add port " + label, zone: zone}
	action.undo = removePortCmd(m.client, zone, port, permanent, action, recordRedo, false)
	action.redo = addPortCmd(m.client, zone, port, permanent, action, recordUndo, false)
	return addPortCmd(m.client, zone, port, permanent, action, recordUndo, true)
}

func (m *Model) actionRemovePort(zone string, port firewalld.Port, permanent bool) tea.Cmd {
	label := port.Port + "/" + port.Protocol
	action := &undoAction{label: "remove port " + label, zone: zone}
	action.undo = addPortCmd(m.client, zone, port, permanent, action, recordRedo, false)
	action.redo = removePortCmd(m.client, zone, port, permanent, action, recordUndo, false)
	return removePortCmd(m.client, zone, port, permanent, action, recordUndo, true)
}

func (m *Model) actionAddRichRule(zone, rule string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add rich rule", zone: zone}
	action.undo = removeRichRuleCmd(m.client, zone, rule, permanent, action, recordRedo, false)
	action.redo = addRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, false)
	return addRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveRichRule(zone, rule string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove rich rule", zone: zone}
	action.undo = addRichRuleCmd(m.client, zone, rule, permanent, action, recordRedo, false)
	action.redo = removeRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, false)
	return removeRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, true)
}

func (m *Model) actionEditRichRule(zone, oldRule, newRule string, permanent bool) tea.Cmd {
	action := &undoAction{label: "edit rich rule", zone: zone}
	action.undo = updateRichRuleCmd(m.client, zone, newRule, oldRule, permanent, action, recordRedo, false)
	action.redo = updateRichRuleCmd(m.client, zone, oldRule, newRule, permanent, action, recordUndo, false)
	return updateRichRuleCmd(m.client, zone, oldRule, newRule, permanent, action, recordUndo, true)
}

func (m *Model) actionAddInterface(zone, iface string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add interface " + iface, zone: zone}
	action.undo = removeInterfaceCmd(m.client, zone, iface, permanent, action, recordRedo, false)
	action.redo = addInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, false)
	return addInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveInterface(zone, iface string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove interface " + iface, zone: zone}
	action.undo = addInterfaceCmd(m.client, zone, iface, permanent, action, recordRedo, false)
	action.redo = removeInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, false)
	return removeInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, true)
}

func (m *Model) actionAddSource(zone, source string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add source " + source, zone: zone}
	action.undo = removeSourceCmd(m.client, zone, source, permanent, action, recordRedo, false)
	action.redo = addSourceCmd(m.client, zone, source, permanent, action, recordUndo, false)
	return addSourceCmd(m.client, zone, source, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveSource(zone, source string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove source " + source, zone: zone}
	action.undo = addSourceCmd(m.client, zone, source, permanent, action, recordRedo, false)
	action.redo = removeSourceCmd(m.client, zone, source, permanent, action, recordUndo, false)
	return removeSourceCmd(m.client, zone, source, permanent, action, recordUndo, true)
}

func (m *Model) actionMasquerade(zone string, enabled, permanent bool) tea.Cmd {
	state := "off"
	if enabled {
		state = "on"
	}
	action := &undoAction{label: "masquerade " + state, zone: zone}
	action.undo = setMasqueradeCmd(m.client, zone, !enabled, permanent, action, recordRedo, false)
	action.redo = setMasqueradeCmd(m.client, zone, enabled, permanent, action, recordUndo, false)
	return setMasqueradeCmd(m.client, zone, enabled, permanent, action, recordUndo, true)
}

func (m *Model) moveMainSelection(delta int) {
	if m.searchQuery != "" {
		m.moveMatchSelection(delta > 0)
		return
	}
	if m.tab == tabIPSets {
		if len(m.ipsets) == 0 {
			return
		}
		next := m.ipsetIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(m.ipsets) {
			next = len(m.ipsets) - 1
		}
		m.ipsetIndex = next
		return
	}
	current := m.currentData()
	if current == nil {
		return
	}
	switch m.tab {
	case tabServices:
		if len(current.Services) == 0 {
			return
		}
		next := m.serviceIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(current.Services) {
			next = len(current.Services) - 1
		}
		m.serviceIndex = next
	case tabPorts:
		if len(current.Ports) == 0 {
			return
		}
		next := m.portIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(current.Ports) {
			next = len(current.Ports) - 1
		}
		m.portIndex = next
	case tabRich:
		if len(current.RichRules) == 0 {
			return
		}
		next := m.richIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(current.RichRules) {
			next = len(current.RichRules) - 1
		}
		m.richIndex = next
	case tabNetwork:
		items := m.networkItems()
		if len(items) == 0 {
			return
		}
		next := m.networkIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(items) {
			next = len(items) - 1
		}
		m.networkIndex = next
		return
	case tabInfo:
		return
	}
}

func (m *Model) moveMatchSelection(forward bool) {
	matches := m.currentMatchIndices()
	if len(matches) == 0 {
		return
	}
	current := m.currentIndex()
	pos := -1
	for i, idx := range matches {
		if idx == current {
			pos = i
			break
		}
	}
	if pos == -1 {
		m.setCurrentIndex(matches[0])
		return
	}
	if forward {
		pos++
		if pos >= len(matches) {
			pos = 0
		}
	} else {
		pos--
		if pos < 0 {
			pos = len(matches) - 1
		}
	}
	m.setCurrentIndex(matches[pos])
}

func (m *Model) startAddInput() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	m.err = nil
	if m.tab == tabNetwork || m.tab == tabInfo {
		if m.tab == tabNetwork {
			m.err = fmt.Errorf("use i/s/m in Network tab")
		} else {
			m.err = fmt.Errorf("editing not implemented for this tab")
		}
		return nil
	}
	if m.tab == tabIPSets {
		if m.currentIPSetName() == "" {
			m.err = fmt.Errorf("no ipset selected (press n to create)")
			return nil
		}
	}
	m.input.SetValue("")
	m.editRichOld = ""
	switch m.tab {
	case tabServices:
		m.input.Placeholder = "service name"
		m.inputMode = inputAddService
	case tabPorts:
		m.input.Placeholder = "port/proto (e.g. 80/tcp)"
		m.inputMode = inputAddPort
	case tabRich:
		m.input.Placeholder = "rich rule"
		m.inputMode = inputAddRich
	case tabIPSets:
		m.input.Placeholder = "entry (IP/CIDR/etc)"
		m.inputMode = inputAddIPSetEntry
	}
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddZone() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "zone name"
	m.inputMode = inputAddZone
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddIPSet() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if !m.permanent {
		m.err = fmt.Errorf("ipset creation is permanent-only (press P)")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "ipset name [type]"
	m.inputMode = inputAddIPSet
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startManualBackup() tea.Cmd {
	if len(m.zones) == 0 || m.selected >= len(m.zones) {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "backup description (optional)"
	m.inputMode = inputManualBackup
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startDeleteIPSet() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if !m.permanent {
		m.err = fmt.Errorf("ipset deletion is permanent-only (press P)")
		return nil
	}
	name := m.currentIPSetName()
	if name == "" {
		m.err = fmt.Errorf("no ipset selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "type ipset name to delete"
	m.inputMode = inputDeleteIPSet
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startRemoveIPSetEntry() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.currentIPSetName() == "" {
		m.err = fmt.Errorf("no ipset selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "entry to remove"
	m.inputMode = inputRemoveIPSetEntry
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startDeleteZone() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if len(m.zones) == 0 || m.selected >= len(m.zones) {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "type zone name to delete"
	m.inputMode = inputDeleteZone
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddInterface() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.tab != tabNetwork {
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "interface name"
	m.inputMode = inputAddInterface
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddSource() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.tab != tabNetwork {
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "source (IP/CIDR)"
	m.inputMode = inputAddSource
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) toggleMasquerade() tea.Cmd {
	current := m.currentData()
	if current == nil || len(m.zones) == 0 {
		return nil
	}
	zone := m.zones[m.selected]
	enabled := !current.Masquerade
	if m.dryRun {
		state := "on"
		if !enabled {
			state = "off"
		}
		m.setDryRunNotice(fmt.Sprintf("set masquerade %s for zone %s (%s)", state, zone, modeLabel(m.permanent)))
		return nil
	}
	return m.maybeBackup(zone, true, m.actionMasquerade(zone, enabled, m.permanent))
}

func (m *Model) startEditRich() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	current := m.currentData()
	if current == nil || len(current.RichRules) == 0 {
		return nil
	}
	if m.richIndex < 0 || m.richIndex >= len(current.RichRules) {
		return nil
	}
	m.err = nil
	m.editRichOld = current.RichRules[m.richIndex]
	m.input.Placeholder = "rich rule"
	m.input.SetValue(m.editRichOld)
	m.inputMode = inputEditRich
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) submitInput() tea.Cmd {
	if m.inputMode == inputNone {
		return nil
	}
	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		m.err = fmt.Errorf("input cannot be empty")
		return nil
	}

	if m.inputMode == inputPanicConfirm {
		if m.panicCountdown > 0 {
			m.err = fmt.Errorf("wait %ds then press Enter", m.panicCountdown)
			return nil
		}
		if !strings.EqualFold(value, "YES") {
			m.err = fmt.Errorf("type YES to confirm")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		if m.dryRun {
			m.setDryRunNotice("enable panic mode")
			return nil
		}
		m.panicAutoArmed = true
		return enablePanicModeCmd(m.client)
	}

	if m.inputMode == inputExportZone {
		current := m.currentData()
		if current == nil {
			m.err = fmt.Errorf("no data loaded")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		return exportZoneCmd(value, current)
	}

	if m.inputMode == inputImportZone {
		if len(m.zones) == 0 || m.selected >= len(m.zones) {
			m.err = fmt.Errorf("no zone selected")
			return nil
		}
		zone := m.zones[m.selected]
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("import zone from %s into %s", value, zone))
			return nil
		}
		return m.maybeBackup(zone, true, importZoneCmd(m.client, zone, value))
	}

	if m.inputMode == inputAddZone {
		if !isValidZoneName(value) {
			m.err = fmt.Errorf("invalid zone name (use letters, digits, _ or -)")
			return nil
		}
		for _, z := range m.zones {
			if z == value {
				m.err = fmt.Errorf("zone already exists")
				return nil
			}
		}
		m.inputMode = inputNone
		m.input.Blur()
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add zone %s", value))
			return nil
		}
		m.loading = true
		m.err = nil
		m.runtimeInvalid = false
		m.pendingZone = value
		return addZoneCmd(m.client, value)
	}

	if m.inputMode == inputDeleteZone {
		if len(m.zones) == 0 || m.selected >= len(m.zones) {
			m.err = fmt.Errorf("no zone selected")
			return nil
		}
		zone := m.zones[m.selected]
		if value != zone {
			m.err = fmt.Errorf("type zone name to confirm deletion")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("delete zone %s", zone))
			return nil
		}
		m.loading = true
		m.err = nil
		m.runtimeInvalid = false
		m.pendingZone = ""
		return m.maybeBackup(zone, true, removeZoneCmd(m.client, zone))
	}

	if m.inputMode == inputManualBackup {
		if len(m.zones) == 0 || m.selected >= len(m.zones) {
			m.err = fmt.Errorf("no zone selected")
			return nil
		}
		zone := m.zones[m.selected]
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		return createManualBackupCmd(zone, value)
	}

	if m.inputMode == inputAddIPSet {
		name, ipsetType, err := parseIPSetInput(value)
		if err != nil {
			m.err = err
			return nil
		}
		if !isValidZoneName(name) {
			m.err = fmt.Errorf("invalid ipset name (use letters, digits, _ or -)")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add ipset %s (%s)", name, ipsetType))
			return nil
		}
		m.ipsetLoading = true
		return addIPSetCmd(m.client, name, ipsetType)
	}

	if m.inputMode == inputAddIPSetEntry {
		name := m.currentIPSetName()
		if name == "" {
			m.err = fmt.Errorf("no ipset selected")
			return nil
		}
		if strings.TrimSpace(value) == "" {
			m.err = fmt.Errorf("entry cannot be empty")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add ipset entry to %s (%s)", name, modeLabel(m.permanent)))
			return nil
		}
		m.ipsetLoading = true
		return addIPSetEntryCmd(m.client, name, strings.TrimSpace(value), m.permanent)
	}

	if m.inputMode == inputRemoveIPSetEntry {
		name := m.currentIPSetName()
		if name == "" {
			m.err = fmt.Errorf("no ipset selected")
			return nil
		}
		if strings.TrimSpace(value) == "" {
			m.err = fmt.Errorf("entry cannot be empty")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("remove ipset entry from %s (%s)", name, modeLabel(m.permanent)))
			return nil
		}
		m.ipsetLoading = true
		return removeIPSetEntryCmd(m.client, name, strings.TrimSpace(value), m.permanent)
	}

	if m.inputMode == inputDeleteIPSet {
		name := m.currentIPSetName()
		if name == "" {
			m.err = fmt.Errorf("no ipset selected")
			return nil
		}
		if strings.TrimSpace(value) != name {
			m.err = fmt.Errorf("type ipset name to confirm deletion")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("delete ipset %s", name))
			return nil
		}
		m.ipsetLoading = true
		return removeIPSetCmd(m.client, name)
	}

	if m.currentData() == nil || len(m.zones) == 0 {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}

	zone := m.zones[m.selected]
	switch m.tab {
	case tabServices:
		m.inputMode = inputNone
		m.input.Blur()
		if !m.servicesLoading && m.servicesErr == nil && len(m.availableServices) > 0 && !m.serviceExists(value) {
			m.err = fmt.Errorf("unknown service: %s", value)
			return nil
		}
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add service %s to zone %s (%s)", value, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionAddService(zone, value, m.permanent))
	case tabPorts:
		port, err := parsePortInput(value)
		if err != nil {
			m.err = err
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		if m.dryRun {
			label := port.Port + "/" + port.Protocol
			m.setDryRunNotice(fmt.Sprintf("add port %s to zone %s (%s)", label, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionAddPort(zone, port, m.permanent))
	case tabRich:
		switch m.inputMode {
		case inputAddRich:
			m.inputMode = inputNone
			m.input.Blur()
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("add rich rule to zone %s (%s)", zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionAddRichRule(zone, value, m.permanent))
		case inputEditRich:
			oldRule := m.editRichOld
			m.editRichOld = ""
			m.inputMode = inputNone
			m.input.Blur()
			if oldRule == value {
				return nil
			}
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("edit rich rule in zone %s (%s)", zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionEditRichRule(zone, oldRule, value, m.permanent))
		}
		return nil
	case tabNetwork:
		switch m.inputMode {
		case inputAddInterface:
			m.inputMode = inputNone
			m.input.Blur()
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("add interface %s to zone %s (%s)", value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionAddInterface(zone, value, m.permanent))
		case inputAddSource:
			if net.ParseIP(value) == nil {
				if _, _, err := net.ParseCIDR(value); err != nil {
					m.err = fmt.Errorf("invalid source: %s", value)
					return nil
				}
			}
			m.inputMode = inputNone
			m.input.Blur()
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("add source %s to zone %s (%s)", value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionAddSource(zone, value, m.permanent))
		}
		return nil
	default:
		return nil
	}
}

func (m *Model) removeSelected() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.tab == tabIPSets {
		return m.startRemoveIPSetEntry()
	}
	current := m.currentData()
	if current == nil || len(m.zones) == 0 {
		return nil
	}
	zone := m.zones[m.selected]

	switch m.tab {
	case tabServices:
		if len(current.Services) == 0 {
			return nil
		}
		service := current.Services[m.serviceIndex]
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("remove service %s from zone %s (%s)", service, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionRemoveService(zone, service, m.permanent))
	case tabPorts:
		if len(current.Ports) == 0 {
			return nil
		}
		port := current.Ports[m.portIndex]
		if m.dryRun {
			label := port.Port + "/" + port.Protocol
			m.setDryRunNotice(fmt.Sprintf("remove port %s from zone %s (%s)", label, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionRemovePort(zone, port, m.permanent))
	case tabRich:
		if len(current.RichRules) == 0 {
			return nil
		}
		rule := current.RichRules[m.richIndex]
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("remove rich rule from zone %s (%s)", zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionRemoveRichRule(zone, rule, m.permanent))
	case tabNetwork:
		items := m.networkItems()
		if len(items) == 0 {
			return nil
		}
		if m.networkIndex < 0 || m.networkIndex >= len(items) {
			return nil
		}
		item := items[m.networkIndex]
		switch item.kind {
		case "iface":
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("remove interface %s from zone %s (%s)", item.value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionRemoveInterface(zone, item.value, m.permanent))
		case "source":
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("remove source %s from zone %s (%s)", item.value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionRemoveSource(zone, item.value, m.permanent))
		default:
			return nil
		}
	case tabInfo:
		m.err = fmt.Errorf("editing not implemented for this tab")
		return nil
	default:
		return nil
	}
}

func (m *Model) currentData() *firewalld.Zone {
	if m.permanent {
		return m.permanentData
	}
	return m.runtimeData
}

func (m *Model) currentIPSetName() string {
	if len(m.ipsets) == 0 {
		return ""
	}
	if m.ipsetIndex < 0 || m.ipsetIndex >= len(m.ipsets) {
		return ""
	}
	return m.ipsets[m.ipsetIndex]
}

func (m *Model) fetchCurrentIPSetEntries() tea.Cmd {
	if m.tab != tabIPSets {
		return nil
	}
	name := m.currentIPSetName()
	if name == "" {
		return nil
	}
	m.ipsetEntriesLoading = true
	m.ipsetEntriesErr = nil
	m.ipsetEntryName = name
	return fetchIPSetEntriesCmd(m.client, name, m.permanent)
}

func (m *Model) currentIndex() int {
	if m.tab == tabPorts {
		return m.portIndex
	}
	if m.tab == tabRich {
		return m.richIndex
	}
	if m.tab == tabNetwork {
		return m.networkIndex
	}
	if m.tab == tabIPSets {
		return m.ipsetIndex
	}
	if m.tab == tabInfo {
		return 0
	}
	return m.serviceIndex
}

func (m *Model) setCurrentIndex(index int) {
	if m.tab == tabPorts {
		m.portIndex = index
		return
	}
	if m.tab == tabRich {
		m.richIndex = index
		return
	}
	if m.tab == tabNetwork {
		m.networkIndex = index
		return
	}
	if m.tab == tabIPSets {
		m.ipsetIndex = index
		return
	}
	if m.tab == tabInfo {
		return
	}
	m.serviceIndex = index
}

func (m *Model) currentItems() []string {
	if m.tab == tabIPSets {
		return m.ipsets
	}
	current := m.currentData()
	if current == nil {
		return nil
	}
	if m.tab == tabPorts {
		items := make([]string, 0, len(current.Ports))
		for _, p := range current.Ports {
			items = append(items, p.Port+"/"+p.Protocol)
		}
		return items
	}
	if m.tab == tabRich {
		return current.RichRules
	}
	if m.tab == tabNetwork {
		items := m.networkItems()
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, item.value)
		}
		return out
	}
	if m.tab == tabInfo {
		return nil
	}
	return current.Services
}

func (m *Model) currentService() string {
	current := m.currentData()
	if current == nil || len(current.Services) == 0 {
		return ""
	}
	if m.serviceIndex < 0 || m.serviceIndex >= len(current.Services) {
		return ""
	}
	return current.Services[m.serviceIndex]
}

func (m *Model) applyTemplate() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if len(m.zones) == 0 {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}
	current := m.currentData()
	if current == nil {
		m.err = fmt.Errorf("no data loaded")
		return nil
	}
	if m.templateIndex < 0 || m.templateIndex >= len(defaultTemplates) {
		m.err = fmt.Errorf("invalid template selection")
		return nil
	}

	tpl := defaultTemplates[m.templateIndex]
	services := filterMissingServices(tpl.Services, current.Services)
	ports := filterMissingPorts(tpl.Ports, current.Ports)
	if len(services) == 0 && len(ports) == 0 {
		m.err = fmt.Errorf("template already applied")
		return nil
	}

	m.templateMode = false
	if m.dryRun {
		zone := m.zones[m.selected]
		m.setDryRunNotice(fmt.Sprintf("apply template %s to zone %s (%s)", tpl.Name, zone, modeLabel(m.permanent)))
		return nil
	}
	m.loading = true
	m.err = nil
	zone := m.zones[m.selected]
	m.pendingZone = zone
	return m.maybeBackup(zone, true, applyTemplateCmd(m.client, zone, services, ports, m.permanent))
}

func filterMissingServices(template, current []string) []string {
	currentSet := make(map[string]struct{}, len(current))
	for _, s := range current {
		currentSet[s] = struct{}{}
	}
	out := make([]string, 0, len(template))
	for _, s := range template {
		if _, ok := currentSet[s]; !ok {
			out = append(out, s)
		}
	}
	return out
}

func filterMissingPorts(template, current []firewalld.Port) []firewalld.Port {
	currentSet := make(map[string]struct{}, len(current))
	for _, p := range current {
		currentSet[p.Port+"/"+p.Protocol] = struct{}{}
	}
	out := make([]firewalld.Port, 0, len(template))
	for _, p := range template {
		if _, ok := currentSet[p.Port+"/"+p.Protocol]; !ok {
			out = append(out, p)
		}
	}
	return out
}

func (m *Model) currentMatchIndices() []int {
	return matchIndices(m.currentItems(), m.searchQuery)
}

func (m *Model) applySearchSelection() {
	if m.searchQuery == "" {
		return
	}
	matches := m.currentMatchIndices()
	if len(matches) == 0 {
		return
	}
	current := m.currentIndex()
	for _, idx := range matches {
		if idx == current {
			return
		}
	}
	m.setCurrentIndex(matches[0])
}

func matchIndices(items []string, query string) []int {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	indices := make([]int, 0, len(items))
	for i, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *Model) nextTab() {
	switch m.tab {
	case tabServices:
		m.tab = tabPorts
	case tabPorts:
		m.tab = tabRich
	case tabRich:
		m.tab = tabNetwork
	case tabNetwork:
		m.tab = tabIPSets
	case tabIPSets:
		m.tab = tabInfo
	case tabInfo:
		m.tab = tabServices
	}
}

func (m *Model) prevTab() {
	switch m.tab {
	case tabServices:
		m.tab = tabInfo
	case tabPorts:
		m.tab = tabServices
	case tabRich:
		m.tab = tabPorts
	case tabNetwork:
		m.tab = tabRich
	case tabIPSets:
		m.tab = tabNetwork
	case tabInfo:
		m.tab = tabIPSets
	}
}

func parsePortInput(value string) (firewalld.Port, error) {
	input := strings.TrimSpace(value)
	if input == "" {
		return firewalld.Port{}, fmt.Errorf("port input is empty")
	}

	var portStr string
	var proto string
	if strings.Contains(input, "/") {
		parts := strings.SplitN(input, "/", 2)
		portStr = strings.TrimSpace(parts[0])
		proto = strings.TrimSpace(parts[1])
	} else {
		fields := strings.Fields(input)
		if len(fields) != 2 {
			return firewalld.Port{}, fmt.Errorf("use format port/proto or \"port proto\"")
		}
		portStr = fields[0]
		proto = fields[1]
	}

	portNum, err := strconv.Atoi(portStr)
	if err != nil || portNum < 1 || portNum > 65535 {
		return firewalld.Port{}, fmt.Errorf("invalid port: %s", portStr)
	}

	proto = strings.ToLower(proto)
	switch proto {
	case "tcp", "udp", "sctp", "dccp":
	default:
		return firewalld.Port{}, fmt.Errorf("invalid protocol: %s", proto)
	}

	return firewalld.Port{Port: portStr, Protocol: proto}, nil
}

func parseIPSetInput(value string) (string, string, error) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return "", "", fmt.Errorf("ipset name is empty")
	}
	if len(fields) > 2 {
		return "", "", fmt.Errorf("use: name [type]")
	}
	name := fields[0]
	ipsetType := "hash:ip"
	if len(fields) == 2 {
		ipsetType = fields[1]
	}
	return name, ipsetType, nil
}

func isValidZoneName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func indexOfZone(zones []string, zone string) int {
	for i, z := range zones {
		if z == zone {
			return i
		}
	}
	return -1
}

func defaultExportPath(zone string) string {
	ts := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("zone-%s-%s.json", zone, ts)
	if dir, err := backup.Dir(); err == nil {
		base := filepath.Dir(dir)
		return filepath.Join(base, "exports", name)
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "lazyfirewall", "exports", name)
	}
	return name
}

func (m *Model) completePath() {
	raw := m.input.Value()
	if raw == "" {
		m.notice = "Type a path, then press Tab"
		return
	}

	expanded, useTilde, home := expandUserPath(raw)
	dir, base := splitPath(expanded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		m.err = err
		return
	}

	type cand struct {
		name  string
		isDir bool
	}
	cands := make([]cand, 0)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, base) {
			cands = append(cands, cand{name: name, isDir: entry.IsDir()})
		}
	}
	if len(cands) == 0 {
		m.notice = "No matches"
		return
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].name < cands[j].name })
	names := make([]string, 0, len(cands))
	for _, c := range cands {
		if c.isDir {
			names = append(names, c.name+string(os.PathSeparator))
		} else {
			names = append(names, c.name)
		}
	}

	prefix := commonPrefix(names)
	if prefix == "" {
		prefix = base
	}

	newValue := joinPath(dir, prefix)
	if strings.HasSuffix(prefix, string(os.PathSeparator)) && !strings.HasSuffix(newValue, string(os.PathSeparator)) {
		newValue += string(os.PathSeparator)
	}
	if useTilde && home != "" && strings.HasPrefix(newValue, home) {
		newValue = "~" + strings.TrimPrefix(newValue, home)
	}

	m.input.SetValue(newValue)
	m.input.CursorEnd()

	if len(names) > 1 && prefix == base {
		m.notice = "Matches: " + strings.Join(limitList(names, 8), "  ")
	} else {
		m.notice = ""
	}
}

func (m *Model) serviceExists(name string) bool {
	if name == "" {
		return false
	}
	for _, s := range m.availableServices {
		if s == name {
			return true
		}
	}
	return false
}

func (m *Model) completeServiceName() {
	raw := strings.TrimSpace(m.input.Value())
	if raw == "" {
		m.notice = "Type a service name, then press Tab"
		return
	}
	if m.servicesLoading {
		m.notice = "Service list loading..."
		return
	}
	if m.servicesErr != nil {
		m.notice = "Service list unavailable"
		return
	}
	if len(m.availableServices) == 0 {
		m.notice = "No services available"
		return
	}
	matches := make([]string, 0)
	for _, name := range m.availableServices {
		if strings.HasPrefix(name, raw) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		m.notice = "No matches"
		return
	}
	sort.Strings(matches)
	prefix := commonPrefix(matches)
	if prefix == "" {
		prefix = raw
	}
	m.input.SetValue(prefix)
	m.input.CursorEnd()
	if len(matches) > 1 && prefix == raw {
		m.notice = "Matches: " + strings.Join(limitList(matches, 8), "  ")
	} else {
		m.notice = ""
	}
}

func expandUserPath(path string) (expanded string, useTilde bool, home string) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			useTilde = true
			home = h
			if path == "~" {
				return h, true, h
			}
			return filepath.Join(h, strings.TrimPrefix(path, "~/")), true, h
		}
	}
	return path, false, ""
}

func splitPath(path string) (dir string, base string) {
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		return path, ""
	}
	dir = filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	return dir, filepath.Base(path)
}

func joinPath(dir, base string) string {
	if dir == "." || dir == "" {
		return base
	}
	return filepath.Join(dir, base)
}

func commonPrefix(items []string) string {
	if len(items) == 0 {
		return ""
	}
	prefix := items[0]
	for _, item := range items[1:] {
		for !strings.HasPrefix(item, prefix) {
			if prefix == "" {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func limitList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	out := append([]string{}, items[:max]...)
	out = append(out, "…")
	return out
}

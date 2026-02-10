//go:build linux
// +build linux

package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if next, cmd, handled := m.handleHelpMode(msg); handled {
		return next, cmd
	}

	if next, cmd, handled := m.handleTemplateMode(msg); handled {
		return next, cmd
	}

	if next, cmd, handled := m.handleBackupMode(msg); handled {
		return next, cmd
	}

	if next, cmd, handled := m.handleInputMode(msg); handled {
		return next, cmd
	}

	if next, cmd, handled := m.handleDetailsMode(msg); handled {
		return next, cmd
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
			if m.logMode {
				m.err = fmt.Errorf("split view not available in logs")
				return m, nil
			}
			if m.tab == tabIPSets {
				m.err = fmt.Errorf("split view not available for IPSets")
				return m, nil
			}
			m.splitView = !m.splitView
			return m, nil
		case "L":
			return m, m.toggleLogs()
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
				m.setDryRunNotice(fmt.Sprintf("commit runtime -> permanent for zone %s", zone))
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
	case logStreamMsg:
		m.logLoading = false
		if msg.err != nil {
			m.logErr = msg.err
			return m, nil
		}
		m.logErr = nil
		m.logLineCh = msg.lines
		m.logCancel = msg.cancel
		return m, readLogLineCmd(msg.lines)
	case logLineMsg:
		if !m.logMode {
			return m, nil
		}
		if logMatchesSource(msg.line) && logMatchesZone(msg.line, m.logZone) {
			m.appendLogLine(msg.line)
		}
		if m.logLineCh != nil {
			return m, readLogLineCmd(m.logLineCh)
		}
		return m, nil
	case logStreamEndMsg:
		m.logLineCh = nil
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

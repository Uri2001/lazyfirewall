//go:build linux
// +build linux

package ui

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

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
		subscribeSignalsCmd(m.client),
		cacheTickCmd(m.cacheTick),
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

	if m.inputMode != inputNone {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
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
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.inputMode == inputSearch {
			m.searchQuery = m.input.Value()
			m.applySearchSelection()
		}
		return m, cmd
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
			m.tab = tabInfo
			return m, nil
		case "h", "left":
			m.detailsMode = false
			m.prevTab()
			return m, nil
		case "l", "right":
			m.detailsMode = false
			m.nextTab()
			return m, nil
		case "S":
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
			if m.searchQuery != "" && !m.splitView && m.focus == focusMain {
				m.moveMatchSelection(true)
			}
			return m, nil
		case "N":
			if m.searchQuery != "" && !m.splitView && m.focus == focusMain {
				m.moveMatchSelection(false)
			}
			return m, nil
		case "r":
			m.loading = true
			m.err = nil
			m.runtimeDenied = false
			m.permanentDenied = false
			m.runtimeInvalid = false
			m.runtimeData = nil
			m.permanentData = nil
			m.editRichOld = ""
			m.cacheInvalidateAll()
			return m, tea.Batch(fetchZonesCmd(m.client), fetchDefaultZoneCmd(m.client), fetchActiveZonesCmd(m.client))
		case "c":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.zones) == 0 {
				return m, nil
			}
			m.loading = true
			m.err = nil
			m.pendingZone = m.zones[m.selected]
			return m, commitRuntimeCmd(m.client, m.zones[m.selected])
		case "u":
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.zones) == 0 {
				return m, nil
			}
			m.loading = true
			m.err = nil
			m.pendingZone = m.zones[m.selected]
			return m, reloadCmd(m.client, m.zones[m.selected])
		case "P":
			m.permanent = !m.permanent
			if len(m.zones) > 0 && m.selected < len(m.zones) {
				m.err = nil
				return m, m.startZoneLoad(m.zones[m.selected], false)
			}
			return m, nil
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
			if m.focus != focusZones {
				return m, nil
			}
			if m.readOnly {
				m.err = firewalld.ErrPermissionDenied
				return m, nil
			}
			if len(m.zones) == 0 || m.selected >= len(m.zones) {
				return m, nil
			}
			zone := m.zones[m.selected]
			m.err = nil
			return m, setDefaultZoneCmd(m.client, zone)
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
		if msg.event.Zone != "" {
			m.cacheInvalidate(msg.event.Zone)
		} else {
			m.cacheInvalidateAll()
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
			listenSignalsCmd(m.signals),
		)
	case cacheTickMsg:
		cmd := cacheTickCmd(m.cacheTick)
		if m.loading {
			return m, cmd
		}
		if len(m.zones) == 0 || m.selected >= len(m.zones) {
			return m, cmd
		}
		zone := m.zones[m.selected]
		cmds := []tea.Cmd{cmd}
		runtimeStale := m.cacheStale(zone, false)
		permanentStale := m.cacheStale(zone, true)
		if runtimeStale {
			cmds = append(cmds, fetchZoneSettingsCmd(m.client, zone, false))
		}
		if permanentStale {
			cmds = append(cmds, fetchZoneSettingsCmd(m.client, zone, true))
		}
		if (!m.permanent && runtimeStale) || (m.permanent && permanentStale) {
			m.loading = true
		}
		return m, tea.Batch(cmds...)
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
			m.cacheSet(msg.zoneName, true, msg.zone)
		} else {
			m.runtimeData = msg.zone
			m.runtimeDenied = false
			m.runtimeInvalid = false
			m.cacheSet(msg.zoneName, false, msg.zone)
		}
		if msg.permanent == m.permanent {
			m.loading = false
		}
		m.clampSelections()
		return m, nil
	case mutationMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.cacheInvalidate(msg.zone)
		m.loading = true
		m.err = nil
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
}

func (m *Model) startZoneLoad(zone string, reset bool) tea.Cmd {
	m.loading = true
	m.pendingZone = zone
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
	}

	runtimeData, runtimeOk := m.cacheGet(zone, false)
	permanentData, permanentOk := m.cacheGet(zone, true)

	if runtimeOk {
		m.runtimeData = runtimeData
		m.runtimeDenied = false
		m.runtimeInvalid = false
	} else if reset {
		m.runtimeData = nil
	}
	if permanentOk {
		m.permanentData = permanentData
		m.permanentDenied = false
	} else if reset {
		m.permanentData = nil
	}

	cmds := make([]tea.Cmd, 0, 2)
	if !runtimeOk {
		cmds = append(cmds, fetchZoneSettingsCmd(m.client, zone, false))
	}
	if !permanentOk {
		cmds = append(cmds, fetchZoneSettingsCmd(m.client, zone, true))
	}

	if (m.permanent && permanentOk) || (!m.permanent && runtimeOk) {
		m.loading = false
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) moveMainSelection(delta int) {
	current := m.currentData()
	if current == nil {
		return
	}
	if m.searchQuery != "" {
		m.moveMatchSelection(delta > 0)
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
	return setMasqueradeCmd(m.client, zone, enabled, m.permanent)
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
		m.loading = true
		m.err = nil
		m.runtimeInvalid = false
		m.pendingZone = ""
		m.cacheInvalidate(zone)
		return removeZoneCmd(m.client, zone)
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
		return addServiceCmd(m.client, zone, value, m.permanent)
	case tabPorts:
		port, err := parsePortInput(value)
		if err != nil {
			m.err = err
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		return addPortCmd(m.client, zone, port, m.permanent)
	case tabRich:
		switch m.inputMode {
		case inputAddRich:
			m.inputMode = inputNone
			m.input.Blur()
			return addRichRuleCmd(m.client, zone, value, m.permanent)
		case inputEditRich:
			oldRule := m.editRichOld
			m.editRichOld = ""
			m.inputMode = inputNone
			m.input.Blur()
			if oldRule == value {
				return nil
			}
			return updateRichRuleCmd(m.client, zone, oldRule, value, m.permanent)
		}
		return nil
	case tabNetwork:
		switch m.inputMode {
		case inputAddInterface:
			m.inputMode = inputNone
			m.input.Blur()
			return addInterfaceCmd(m.client, zone, value, m.permanent)
		case inputAddSource:
			if net.ParseIP(value) == nil {
				if _, _, err := net.ParseCIDR(value); err != nil {
					m.err = fmt.Errorf("invalid source: %s", value)
					return nil
				}
			}
			m.inputMode = inputNone
			m.input.Blur()
			return addSourceCmd(m.client, zone, value, m.permanent)
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
		return removeServiceCmd(m.client, zone, service, m.permanent)
	case tabPorts:
		if len(current.Ports) == 0 {
			return nil
		}
		port := current.Ports[m.portIndex]
		return removePortCmd(m.client, zone, port, m.permanent)
	case tabRich:
		if len(current.RichRules) == 0 {
			return nil
		}
		rule := current.RichRules[m.richIndex]
		return removeRichRuleCmd(m.client, zone, rule, m.permanent)
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
			return removeInterfaceCmd(m.client, zone, item.value, m.permanent)
		case "source":
			return removeSourceCmd(m.client, zone, item.value, m.permanent)
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
	if m.tab == tabInfo {
		return
	}
	m.serviceIndex = index
}

func (m *Model) currentItems() []string {
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
	m.loading = true
	m.err = nil
	zone := m.zones[m.selected]
	m.pendingZone = zone
	return applyTemplateCmd(m.client, zone, services, ports, m.permanent)
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
	case tabInfo:
		m.tab = tabNetwork
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

func (m *Model) cacheGet(zone string, permanent bool) (*firewalld.Zone, bool) {
	entry, ok := m.cache[zone]
	if !ok {
		return nil, false
	}
	now := time.Now()
	if permanent {
		if entry.permanent == nil || now.Sub(entry.permanentAt) > m.cacheTTL {
			return nil, false
		}
		return entry.permanent, true
	}
	if entry.runtime == nil || now.Sub(entry.runtimeAt) > m.cacheTTL {
		return nil, false
	}
	return entry.runtime, true
}

func (m *Model) cacheSet(zone string, permanent bool, data *firewalld.Zone) {
	if zone == "" || data == nil {
		return
	}
	entry, ok := m.cache[zone]
	if !ok {
		entry = &zoneCache{}
		m.cache[zone] = entry
	}
	now := time.Now()
	if permanent {
		entry.permanent = data
		entry.permanentAt = now
	} else {
		entry.runtime = data
		entry.runtimeAt = now
	}
}

func (m *Model) cacheStale(zone string, permanent bool) bool {
	_, ok := m.cacheGet(zone, permanent)
	return !ok
}

func (m *Model) cacheInvalidate(zone string) {
	if zone == "" {
		return
	}
	delete(m.cache, zone)
}

func (m *Model) cacheInvalidateAll() {
	m.cache = make(map[string]*zoneCache)
}

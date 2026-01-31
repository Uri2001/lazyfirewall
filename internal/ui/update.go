package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/models"
)

type zonesMsg struct {
	zones []string
	err   error
}

type activeZonesMsg struct {
	zones map[string][]string
	err   error
}

type defaultZoneMsg struct {
	zone string
	err  error
}

type zoneDataMsg struct {
	data *models.ZoneData
	err  error
	perm bool
}

type serviceDetailsMsg struct {
	name string
	info *firewalld.ServiceInfo
	err  error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % focusCount
		case "shift+tab", "backtab":
			m.focus = (m.focus + focusCount - 1) % focusCount
		case "j", "down":
			if m.focus == focusSidebar && m.selectedZone < len(m.zones)-1 {
				m.selectedZone++
				return m, m.fetchSelectedZone()
			}
			if m.focus == focusMain {
				m, cmd := advanceSelection(m, 1)
				return m, cmd
			}
		case "k", "up":
			if m.focus == focusSidebar && m.selectedZone > 0 {
				m.selectedZone--
				return m, m.fetchSelectedZone()
			}
			if m.focus == focusMain {
				m, cmd := advanceSelection(m, -1)
				return m, cmd
			}
		case "h", "left":
			if m.focus == focusMain {
				m = setTab(m, int(m.tab)-1)
				return m, maybeFetchServiceDetails(m)
			}
		case "l", "right":
			if m.focus == focusMain {
				m = setTab(m, int(m.tab)+1)
				return m, maybeFetchServiceDetails(m)
			}
		case "1", "2", "3", "4", "5":
			if m.focus == focusMain {
				m = setTab(m, int(msg.String()[0]-'1'))
				return m, maybeFetchServiceDetails(m)
			}
		case "D":
			m.debugMode = !m.debugMode
		case "P":
			m.permanent = !m.permanent
			m.zoneData = m.selectActiveData()
			return m, m.fetchSelectedZone()
		case "S":
			m.splitView = !m.splitView
		case "?":
			m.showHelp = !m.showHelp
		case "r":
			return m, tea.Batch(fetchZonesCmd(m.client), fetchActiveZonesCmd(m.client), fetchDefaultZoneCmd(m.client), m.fetchSelectedZone())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case zonesMsg:
		m.zones = msg.zones
		m.err = msg.err
		if m.err == nil && len(m.zones) > 0 && m.zoneData == nil {
			return m, m.fetchSelectedZone()
		}

	case activeZonesMsg:
		m.activeZones = msg.zones
		if msg.err != nil {
			m.err = msg.err
		}

	case defaultZoneMsg:
		m.defaultZone = msg.zone
		if msg.err != nil {
			m.err = msg.err
		}

	case zoneDataMsg:
		m.loading = false
		if msg.perm {
			m.permaData = msg.data
		} else {
			m.runtimeData = msg.data
		}
		m.zoneData = m.selectActiveData()
		if m.zoneData != nil {
			m.selectedService = clampIndex(m.selectedService, len(m.zoneData.Services))
			m.selectedPort = clampIndex(m.selectedPort, len(m.zoneData.Ports))
			m.selectedRule = clampIndex(m.selectedRule, len(m.zoneData.RichRules))
		}
		if msg.err != nil {
			m.err = msg.err
		}
		return m, maybeFetchServiceDetails(m)

	case serviceDetailsMsg:
		m.serviceLoading[msg.name] = false
		if msg.err != nil {
			m.serviceDetailsErr[msg.name] = msg.err
		} else {
			m.serviceDetails[msg.name] = msg.info
		}
	}

	return m, nil
}

func (m Model) fetchSelectedZone() tea.Cmd {
	if m.selectedZone < 0 || m.selectedZone >= len(m.zones) {
		return nil
	}
	m.loading = true
	zone := m.zones[m.selectedZone]
	return tea.Batch(
		fetchZoneDataCmd(m.client, zone, false),
		fetchZoneDataCmd(m.client, zone, true),
	)
}

func (m Model) selectActiveData() *models.ZoneData {
	if m.permanent {
		if m.permaData != nil {
			return m.permaData
		}
		return m.zoneData
	}
	if m.runtimeData != nil {
		return m.runtimeData
	}
	return m.zoneData
}

func setTab(m Model, idx int) Model {
	if idx < 0 {
		idx = int(tabInfo)
	}
	if idx > int(tabInfo) {
		idx = 0
	}
	m.tab = mainTab(idx)
	return m
}

func advanceSelection(m Model, delta int) (Model, tea.Cmd) {
	switch m.tab {
	case tabServices:
		max := 0
		if m.zoneData != nil {
			max = len(m.zoneData.Services)
		}
		m.selectedService = clampIndex(m.selectedService+delta, max)
		return m, maybeFetchServiceDetails(m)
	case tabPorts:
		max := 0
		if m.zoneData != nil {
			max = len(m.zoneData.Ports)
		}
		m.selectedPort = clampIndex(m.selectedPort+delta, max)
	case tabRules:
		max := 0
		if m.zoneData != nil {
			max = len(m.zoneData.RichRules)
		}
		m.selectedRule = clampIndex(m.selectedRule+delta, max)
	}
	return m, nil
}

func clampIndex(value, max int) int {
	if max <= 0 {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value >= max {
		return max - 1
	}
	return value
}

func maybeFetchServiceDetails(m Model) tea.Cmd {
	if m.tab != tabServices || m.zoneData == nil || m.selectedService >= len(m.zoneData.Services) {
		return nil
	}
	name := m.zoneData.Services[m.selectedService]
	if name == "" {
		return nil
	}
	if m.serviceDetails[name] != nil || m.serviceLoading[name] {
		return nil
	}
	m.serviceLoading[name] = true
	return fetchServiceDetailsCmd(m.client, name)
}

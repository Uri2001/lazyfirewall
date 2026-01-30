package ui

import (
	tea "github.com/charmbracelet/bubbletea"

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
				m = advanceSelection(m, 1)
			}
		case "k", "up":
			if m.focus == focusSidebar && m.selectedZone > 0 {
				m.selectedZone--
				return m, m.fetchSelectedZone()
			}
			if m.focus == focusMain {
				m = advanceSelection(m, -1)
			}
		case "h", "left":
			if m.focus == focusMain {
				m = setTab(m, int(m.tab)-1)
			}
		case "l", "right":
			if m.focus == focusMain {
				m = setTab(m, int(m.tab)+1)
			}
		case "1", "2", "3", "4", "5":
			if m.focus == focusMain {
				m = setTab(m, int(msg.String()[0]-'1'))
			}
		case "r":
			return m, tea.Batch(fetchZonesCmd(m.client), fetchActiveZonesCmd(m.client), fetchDefaultZoneCmd(m.client))
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
		m.zoneData = msg.data
		if msg.err != nil {
			m.err = msg.err
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
	return fetchZoneDataCmd(m.client, zone)
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

func advanceSelection(m Model, delta int) Model {
	switch m.tab {
	case tabServices:
		max := 0
		if m.zoneData != nil {
			max = len(m.zoneData.Services)
		}
		m.selectedService = clampIndex(m.selectedService+delta, max)
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
	return m
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

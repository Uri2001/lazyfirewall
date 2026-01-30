package ui

import (
	tea "github.com/charmbracelet/bubbletea"
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % 3
		case "shift+tab":
			m.focus = (m.focus + 2) % 3
		case "j", "down":
			if m.focus == focusSidebar && m.selectedZone < len(m.zones)-1 {
				m.selectedZone++
			}
		case "k", "up":
			if m.focus == focusSidebar && m.selectedZone > 0 {
				m.selectedZone--
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
	}

	return m, nil
}

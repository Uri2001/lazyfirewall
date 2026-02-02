//go:build linux
// +build linux

package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchZonesCmd(m.client))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "P":
			m.permanent = !m.permanent
			if len(m.zones) > 0 && m.selected < len(m.zones) {
				m.loading = true
				m.err = nil
				m.pendingZone = m.zones[m.selected]
				return m, fetchZoneSettingsCmd(m.client, m.zones[m.selected], m.permanent)
			}
			return m, nil
		case "r":
			m.loading = true
			m.err = nil
			return m, fetchZonesCmd(m.client)
		case "j", "down":
			if m.focus == focusZones && len(m.zones) > 0 && m.selected < len(m.zones)-1 {
				m.selected++
				m.loading = true
				m.err = nil
				m.pendingZone = m.zones[m.selected]
				return m, fetchZoneSettingsCmd(m.client, m.zones[m.selected], m.permanent)
			}
			return m, nil
		case "k", "up":
			if m.focus == focusZones && len(m.zones) > 0 && m.selected > 0 {
				m.selected--
				m.loading = true
				m.err = nil
				m.pendingZone = m.zones[m.selected]
				return m, fetchZoneSettingsCmd(m.client, m.zones[m.selected], m.permanent)
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
		if m.selected >= len(m.zones) {
			m.selected = 0
		}
		m.loading = true
		m.pendingZone = m.zones[m.selected]
		return m, fetchZoneSettingsCmd(m.client, m.zones[m.selected], m.permanent)
	case zoneSettingsMsg:
		if msg.zoneName != "" && msg.zoneName != m.pendingZone {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.zoneData = msg.zone
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

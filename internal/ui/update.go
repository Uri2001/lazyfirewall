//go:build linux
// +build linux

package ui

import (
	"fmt"
	"strconv"
	"strings"

	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchZonesCmd(m.client))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.inputMode != inputNone {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.inputMode = inputNone
				m.input.Blur()
				return m, nil
			case "enter":
				return m, m.submitInput()
			}
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
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
		case "1", "h", "left":
			m.tab = tabServices
			return m, nil
		case "2", "l", "right":
			m.tab = tabPorts
			return m, nil
		case "r":
			m.loading = true
			m.err = nil
			return m, fetchZonesCmd(m.client)
		case "c":
			if len(m.zones) == 0 {
				return m, nil
			}
			m.loading = true
			m.err = nil
			m.pendingZone = m.zones[m.selected]
			return m, commitRuntimeCmd(m.client, m.zones[m.selected])
		case "u":
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
				m.loading = true
				m.err = nil
				m.pendingZone = m.zones[m.selected]
				return m, tea.Batch(
					fetchZoneSettingsCmd(m.client, m.zones[m.selected], false),
					fetchZoneSettingsCmd(m.client, m.zones[m.selected], true),
				)
			}
			return m, nil
		case "j", "down":
			if m.focus == focusZones {
				if len(m.zones) > 0 && m.selected < len(m.zones)-1 {
					m.selected++
					m.loading = true
					m.err = nil
					m.pendingZone = m.zones[m.selected]
					return m, tea.Batch(
						fetchZoneSettingsCmd(m.client, m.zones[m.selected], false),
						fetchZoneSettingsCmd(m.client, m.zones[m.selected], true),
					)
				}
				return m, nil
			}
			m.moveMainSelection(1)
			return m, nil
		case "k", "up":
			if m.focus == focusZones {
				if len(m.zones) > 0 && m.selected > 0 {
					m.selected--
					m.loading = true
					m.err = nil
					m.pendingZone = m.zones[m.selected]
					return m, tea.Batch(
						fetchZoneSettingsCmd(m.client, m.zones[m.selected], false),
						fetchZoneSettingsCmd(m.client, m.zones[m.selected], true),
					)
				}
				return m, nil
			}
			m.moveMainSelection(-1)
			return m, nil
		case "a":
			if m.focus == focusMain {
				return m, m.startAddInput()
			}
			return m, nil
		case "d":
			if m.focus == focusMain {
				return m, m.removeSelected()
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
		return m, tea.Batch(
			fetchZoneSettingsCmd(m.client, m.zones[m.selected], false),
			fetchZoneSettingsCmd(m.client, m.zones[m.selected], true),
		)
	case zoneSettingsMsg:
		if msg.zoneName != "" && msg.zoneName != m.pendingZone {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if msg.permanent {
			m.permanentData = msg.zone
		} else {
			m.runtimeData = msg.zone
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
		m.loading = true
		m.err = nil
		m.pendingZone = msg.zone
		return m, tea.Batch(
			fetchZoneSettingsCmd(m.client, msg.zone, false),
			fetchZoneSettingsCmd(m.client, msg.zone, true),
		)
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
}

func (m *Model) moveMainSelection(delta int) {
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
	}
}

func (m *Model) startAddInput() tea.Cmd {
	m.err = nil
	m.input.SetValue("")
	switch m.tab {
	case tabServices:
		m.input.Placeholder = "service name"
		m.inputMode = inputAddService
	case tabPorts:
		m.input.Placeholder = "port/proto (e.g. 80/tcp)"
		m.inputMode = inputAddPort
	}
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) submitInput() tea.Cmd {
	if m.inputMode == inputNone {
		return nil
	}
	if m.currentData() == nil || len(m.zones) == 0 {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}

	zone := m.zones[m.selected]
	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		m.err = fmt.Errorf("input cannot be empty")
		return nil
	}

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
	default:
		return nil
	}
}

func (m *Model) removeSelected() tea.Cmd {
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

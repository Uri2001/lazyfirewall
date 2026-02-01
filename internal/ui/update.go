package ui

import (
	"errors"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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

type mutationMsg struct {
	notice string
	err    error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inputMode != inputNone {
			return m.handleInput(msg)
		}
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
		case "a":
			if m.focus == focusMain {
				if m.tab == tabServices {
					return m.startInput(inputAddService, "Add service")
				}
				if m.tab == tabPorts {
					return m.startInput(inputAddPort, "Add port (e.g. 8080/tcp)")
				}
			}
		case "d":
			if m.focus == focusMain {
				return m.handleDelete()
			}
		case " ":
			if m.focus == focusMain && m.tab == tabServices {
				return m.handleDelete()
			}
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

	case mutationMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.status = msg.notice
			return m, m.fetchSelectedZone()
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

func (m Model) startInput(mode inputMode, placeholder string) (Model, tea.Cmd) {
	m.inputMode = mode
	m.inputErr = ""
	m.textInput.SetValue("")
	m.textInput.Placeholder = placeholder
	m.textInput.Focus()
	return m, textinput.Blink
}

func (m Model) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = inputNone
		m.inputErr = ""
		m.textInput.Blur()
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.textInput.Value())
		if value == "" {
			m.inputErr = "value required"
			return m, nil
		}
		m.textInput.Blur()
		mode := m.inputMode
		m.inputMode = inputNone
		switch mode {
		case inputAddService:
			return m, addServiceCmd(m.client, m.currentZone(), value, m.permanent)
		case inputAddPort:
			port, err := parsePortInput(value)
			if err != nil {
				m.inputErr = err.Error()
				return m, nil
			}
			return m, addPortCmd(m.client, m.currentZone(), port, m.permanent)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	switch m.tab {
	case tabServices:
		if m.zoneData == nil || len(m.zoneData.Services) == 0 {
			return m, nil
		}
		service := m.zoneData.Services[m.selectedService]
		return m, removeServiceCmd(m.client, m.currentZone(), service, m.permanent)
	case tabPorts:
		if m.zoneData == nil || len(m.zoneData.Ports) == 0 {
			return m, nil
		}
		port := m.zoneData.Ports[m.selectedPort]
		return m, removePortCmd(m.client, m.currentZone(), port, m.permanent)
	default:
		return m, nil
	}
}

func (m Model) currentZone() string {
	if m.selectedZone < 0 || m.selectedZone >= len(m.zones) {
		return ""
	}
	return m.zones[m.selectedZone]
}

func parsePortInput(value string) (firewalld.Port, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return firewalld.Port{}, errors.New("port is empty")
	}
	if strings.Contains(raw, "/") {
		p, err := firewalld.ParsePortString(raw)
		if err != nil {
			return firewalld.Port{}, err
		}
		return validatePort(p)
	}

	parts := strings.Fields(raw)
	if len(parts) != 2 {
		return firewalld.Port{}, errors.New("expected format: 8080/tcp or '8080 tcp'")
	}
	var port firewalld.Port
	if n, err := strconv.Atoi(parts[0]); err == nil {
		port = firewalld.Port{Number: n, Protocol: parts[1]}
	} else if n, err := strconv.Atoi(parts[1]); err == nil {
		port = firewalld.Port{Number: n, Protocol: parts[0]}
	} else {
		return firewalld.Port{}, errors.New("invalid port number")
	}
	return validatePort(port)
}

func validatePort(port firewalld.Port) (firewalld.Port, error) {
	if port.Number <= 0 || port.Number > 65535 {
		return firewalld.Port{}, errors.New("port must be 1-65535")
	}
	switch port.Protocol {
	case "tcp", "udp", "sctp", "dccp":
		return port, nil
	default:
		return firewalld.Port{}, errors.New("protocol must be tcp/udp/sctp/dccp")
	}
}

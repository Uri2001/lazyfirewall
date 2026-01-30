package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1)

	mainStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	defaultBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("228")).
				Bold(true)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("252"))

	tabActiveStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("170"))
)

func (m Model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	sidebar := m.renderSidebar()
	main := m.renderMain()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main),
		footer,
	)
}

func (m Model) renderSidebar() string {
	var b strings.Builder
	b.WriteString("Zones\n\n")

	for i, zone := range m.zones {
		isActive := len(m.activeZones[zone]) > 0
		isDefault := zone == m.defaultZone

		prefix := "  "
		if isActive {
			prefix = activeStyle.Render("● ")
		}

		name := zone
		if i == m.selectedZone {
			name = selectedStyle.Render(name)
		}

		line := prefix + name
		if isDefault {
			line += " " + defaultBadgeStyle.Render("[D]")
		}
		b.WriteString(line + "\n")
	}

	return sidebarStyle.Render(b.String())
}

func (m Model) renderMain() string {
	if len(m.zones) == 0 {
		return mainStyle.Render("Loading...")
	}
	zone := m.zones[m.selectedZone]
	if m.loading {
		return mainStyle.Render("Loading zone data...")
	}
	if m.zoneData == nil {
		return mainStyle.Render("No data for zone: " + zone)
	}
	if m.debugMode {
		return mainStyle.Render(m.renderDebug())
	}

	var b strings.Builder
	b.WriteString("Selected: " + zone + "\n\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	switch m.tab {
	case tabServices:
		b.WriteString(m.renderServices())
	case tabPorts:
		b.WriteString(m.renderPorts())
	case tabRules:
		b.WriteString(m.renderRules())
	case tabMasquerade:
		b.WriteString(m.renderMasquerade())
	case tabInfo:
		b.WriteString(m.renderInfo())
	}

	return mainStyle.Render(b.String())
}

func (m Model) renderFooter() string {
	return "[q] Quit  [tab] Switch Panel  [↑↓] Navigate  [h/l] Tabs  [1-5] Jump  [D] Debug  [r] Refresh"
}

func (m Model) renderTabs() string {
	tabs := []string{"Services", "Ports", "Rich Rules", "Masquerade", "Info"}
	var parts []string
	for i, tab := range tabs {
		style := tabStyle
		if mainTab(i) == m.tab {
			style = tabActiveStyle
		}
		parts = append(parts, style.Render(tab))
	}
	return strings.Join(parts, "")
}

func (m Model) renderServices() string {
	if len(m.zoneData.Services) == 0 {
		return "(no services)"
	}
	var b strings.Builder
	for i, service := range m.zoneData.Services {
		line := "  " + service
		if i == m.selectedService && m.focus == focusMain {
			line = selectedStyle.Render("› " + service)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) renderPorts() string {
	if len(m.zoneData.Ports) == 0 {
		return "(no ports)"
	}
	var b strings.Builder
	for i, port := range m.zoneData.Ports {
		line := "  " + port.Protocol + " " + itoa(port.Number)
		if i == m.selectedPort && m.focus == focusMain {
			line = selectedStyle.Render("› " + port.Protocol + " " + itoa(port.Number))
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) renderRules() string {
	if len(m.zoneData.RichRules) == 0 {
		return "(no rich rules)"
	}
	var b strings.Builder
	for i, rule := range m.zoneData.RichRules {
		line := "  " + rule
		if i == m.selectedRule && m.focus == focusMain {
			line = selectedStyle.Render("› " + rule)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) renderMasquerade() string {
	status := "OFF"
	if m.zoneData.Masquerade {
		status = "ON"
	}
	var b strings.Builder
	b.WriteString("Masquerade: " + status + "\n\n")
	b.WriteString("Interfaces:\n")
	if len(m.zoneData.Interfaces) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, iface := range m.zoneData.Interfaces {
			b.WriteString("  • " + iface + "\n")
		}
	}
	b.WriteString("\nSources:\n")
	if len(m.zoneData.Sources) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, src := range m.zoneData.Sources {
			b.WriteString("  • " + src + "\n")
		}
	}
	return b.String()
}

func (m Model) renderInfo() string {
	var b strings.Builder
	b.WriteString("Zone: " + m.zoneData.Zone + "\n")
	b.WriteString("Interfaces: " + strings.Join(m.zoneData.Interfaces, ", ") + "\n")
	b.WriteString("Sources: " + strings.Join(m.zoneData.Sources, ", ") + "\n")
	return b.String()
}

func (m Model) renderDebug() string {
	var b strings.Builder
	b.WriteString("Debug: getZoneSettings2\n\n")
	if len(m.zoneData.RawKeys) > 0 {
		b.WriteString("Keys:\n")
		for _, key := range m.zoneData.RawKeys {
			b.WriteString("  • " + key + "\n")
		}
		b.WriteString("\n")
	}
	if len(m.zoneData.RawPorts) > 0 {
		b.WriteString("Ports raw:\n")
		for _, line := range m.zoneData.RawPorts {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}
	if len(m.zoneData.RawDump) > 0 {
		b.WriteString("Dump:\n")
		for _, line := range m.zoneData.RawDump {
			b.WriteString("  " + line + "\n")
		}
	}
	return b.String()
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

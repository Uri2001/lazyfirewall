package ui

import (
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
	content := "Selected: " + zone + "\n\n[Details will go here]"
	return mainStyle.Render(content)
}

func (m Model) renderFooter() string {
	return "[q] Quit  [tab] Switch Panel  [↑↓] Navigate  [r] Refresh"
}

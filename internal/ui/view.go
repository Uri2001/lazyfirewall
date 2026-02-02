//go:build linux
// +build linux

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	statusStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250")).Padding(0, 1)
	sidebarStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	mainStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
)

func (m Model) View() string {
	sidebarWidth := 24
	if m.width > 0 {
		if m.width/4 > sidebarWidth {
			sidebarWidth = m.width / 4
		}
		if sidebarWidth > 32 {
			sidebarWidth = 32
		}
	}

	mainWidth := 80
	if m.width > 0 {
		mainWidth = m.width - sidebarWidth - 1
		if mainWidth < 40 {
			mainWidth = 40
		}
	}

	sidebar := renderSidebar(m, sidebarWidth)
	main := renderMain(m, mainWidth)
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	status := renderStatus(m)
	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

func renderSidebar(m Model, width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Zones"))
	b.WriteString("\n")

	if len(m.zones) == 0 {
		b.WriteString(dimStyle.Render("No zones"))
		return sidebarStyle.Width(width).Render(b.String())
	}

	for i, zone := range m.zones {
		prefix := "  "
		line := zone
		if i == m.selected {
			prefix = "› "
			if m.focus == focusZones {
				line = selectedStyle.Render(zone)
			} else {
				line = titleStyle.Render(zone)
			}
		}
		b.WriteString(prefix + line + "\n")
	}

	return sidebarStyle.Width(width).Render(b.String())
}

func renderMain(m Model, width int) string {
	var b strings.Builder

	zoneName := "None"
	if len(m.zones) > 0 && m.selected < len(m.zones) {
		zoneName = m.zones[m.selected]
	}

	mode := "Runtime"
	if m.permanent {
		mode = "Permanent"
	}

	header := fmt.Sprintf("%s (%s)", zoneName, mode)
	if m.loading {
		header = fmt.Sprintf("%s %s Loading...", header, m.spinner.View())
	}

	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	if m.zoneData == nil {
		b.WriteString(dimStyle.Render("No data loaded"))
		return mainStyle.Width(width).Render(b.String())
	}

	b.WriteString(titleStyle.Render("Services"))
	b.WriteString("\n")
	if len(m.zoneData.Services) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
	} else {
		for _, s := range m.zoneData.Services {
			b.WriteString("  - " + s + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Ports"))
	b.WriteString("\n")
	if len(m.zoneData.Ports) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
	} else {
		for _, p := range m.zoneData.Ports {
			b.WriteString(fmt.Sprintf("  - %s/%s\n", p.Port, p.Protocol))
		}
	}

	return mainStyle.Width(width).Render(b.String())
}

func renderStatus(m Model) string {
	mode := "Runtime"
	if m.permanent {
		mode = "Permanent"
	}
	status := fmt.Sprintf("Mode: %s | Tab: focus  j/k: move  r: refresh  q: quit", mode)
	return statusStyle.Render(status)
}

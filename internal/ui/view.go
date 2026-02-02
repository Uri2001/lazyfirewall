//go:build linux
// +build linux

package ui

import (
	"fmt"
	"strings"

	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle       = lipgloss.NewStyle().Bold(true)
	selectedStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	tabActiveStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Padding(0, 1)
	tabInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("237")).Padding(0, 1)
	inputStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("229"))
	statusStyle      = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250")).Padding(0, 1)
	sidebarStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	mainStyle        = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
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

	current := m.currentData()
	if current == nil {
		b.WriteString(dimStyle.Render("No data loaded"))
		return mainStyle.Width(width).Render(b.String())
	}

	b.WriteString("\n")
	b.WriteString(renderTabs(m))
	b.WriteString("\n\n")
	switch m.tab {
	case tabServices:
		renderServicesList(&b, m, current)
	case tabPorts:
		renderPortsList(&b, m, current)
	}

	if m.inputMode != inputNone {
		b.WriteString("\n")
		b.WriteString(renderInput(m))
	}

	return mainStyle.Width(width).Render(b.String())
}

func renderTabs(m Model) string {
	serviceLabel := " Services "
	portLabel := " Ports "
	if m.tab == tabServices {
		serviceLabel = tabActiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
	} else {
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabActiveStyle.Render(portLabel)
	}
	return serviceLabel + " " + portLabel
}

func renderServicesList(b *strings.Builder, m Model, current *firewalld.Zone) {
	if len(current.Services) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		return
	}

	permanentSet := make(map[string]struct{})
	if !m.permanent && m.permanentData != nil {
		for _, s := range m.permanentData.Services {
			permanentSet[s] = struct{}{}
		}
	}

	for i, s := range current.Services {
		prefix := "  "
		line := s
		if !m.permanent && m.permanentData != nil {
			if _, ok := permanentSet[s]; !ok {
				line = s + " *"
			}
		}
		if i == m.serviceIndex {
			prefix = "› "
			if m.focus == focusMain {
				line = selectedStyle.Render(line)
			} else {
				line = titleStyle.Render(line)
			}
		}
		b.WriteString(prefix + line + "\n")
	}
}

func renderPortsList(b *strings.Builder, m Model, current *firewalld.Zone) {
	if len(current.Ports) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		return
	}

	permanentSet := make(map[string]struct{})
	if !m.permanent && m.permanentData != nil {
		for _, p := range m.permanentData.Ports {
			key := p.Port + "/" + p.Protocol
			permanentSet[key] = struct{}{}
		}
	}

	for i, p := range current.Ports {
		line := fmt.Sprintf("%s/%s", p.Port, p.Protocol)
		prefix := "  "
		if !m.permanent && m.permanentData != nil {
			if _, ok := permanentSet[line]; !ok {
				line = line + " *"
			}
		}
		if i == m.portIndex {
			prefix = "› "
			if m.focus == focusMain {
				line = selectedStyle.Render(line)
			} else {
				line = titleStyle.Render(line)
			}
		}
		b.WriteString(prefix + line + "\n")
	}
}

func renderInput(m Model) string {
	label := ""
	mode := "runtime"
	if m.permanent {
		mode = "permanent"
	}
	switch m.inputMode {
	case inputAddService:
		label = "Add service (" + mode + "): "
	case inputAddPort:
		label = "Add port (" + mode + "): "
	}
	return inputStyle.Render(label) + m.input.View()
}

func renderStatus(m Model) string {
	mode := "Runtime"
	if m.permanent {
		mode = "Permanent"
	}
	legend := ""
	if !m.permanent {
		legend = " | * runtime-only"
	}
	status := fmt.Sprintf("Mode: %s | 1/2: tabs  a: add  d: delete  c: commit  u: revert  Tab: focus  j/k: move  P: toggle  r: refresh  q: quit%s", mode, legend)
	return statusStyle.Render(status)
}

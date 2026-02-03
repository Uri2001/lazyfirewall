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
	matchStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
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
	if m.splitView {
		b.WriteString(renderSplitView(m, width))
	} else {
		switch m.tab {
		case tabServices:
			renderServicesList(&b, m, current)
		case tabPorts:
			renderPortsList(&b, m, current)
		case tabRich:
			renderRichRulesList(&b, m, current)
		case tabNetwork:
			renderNetworkView(&b, m, current)
		}
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
	richLabel := " Rich Rules "
	networkLabel := " Network "
	switch m.tab {
	case tabServices:
		serviceLabel = tabActiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
	case tabPorts:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabActiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
	case tabRich:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabActiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
	case tabNetwork:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabActiveStyle.Render(networkLabel)
	}
	return serviceLabel + " " + portLabel + " " + richLabel + " " + networkLabel
}

func renderSplitView(m Model, width int) string {
	leftWidth := width/2 - 1
	if leftWidth < 20 {
		leftWidth = 20
	}
	rightWidth := width - leftWidth - 1
	if rightWidth < 20 {
		rightWidth = 20
	}

	leftLines, rightLines := splitLines(m)
	left := titleStyle.Render("Runtime") + "\n" + strings.Join(leftLines, "\n")
	right := titleStyle.Render("Permanent") + "\n" + strings.Join(rightLines, "\n")

	leftBox := lipgloss.NewStyle().Width(leftWidth).Render(left)
	rightBox := lipgloss.NewStyle().Width(rightWidth).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

func splitLines(m Model) ([]string, []string) {
	switch m.tab {
	case tabServices:
		return diffServices(m.runtimeData, m.permanentData)
	case tabPorts:
		return diffPorts(m.runtimeData, m.permanentData)
	case tabRich:
		return diffRichRules(m.runtimeData, m.permanentData)
	case tabNetwork:
		return diffNetwork(m.runtimeData, m.permanentData)
	default:
		return []string{""}, []string{""}
	}
}

func diffServices(runtime, permanent *firewalld.Zone) ([]string, []string) {
	if runtime == nil || permanent == nil {
		return []string{dimStyle.Render("(loading)")}, []string{dimStyle.Render("(loading)")}
	}

	permanentSet := make(map[string]struct{}, len(permanent.Services))
	for _, s := range permanent.Services {
		permanentSet[s] = struct{}{}
	}
	runtimeSet := make(map[string]struct{}, len(runtime.Services))
	for _, s := range runtime.Services {
		runtimeSet[s] = struct{}{}
	}

	left := make([]string, 0, len(runtime.Services))
	for _, s := range runtime.Services {
		prefix := "  "
		if _, ok := permanentSet[s]; !ok {
			prefix = "+ "
		}
		left = append(left, prefix+s)
	}

	right := make([]string, 0, len(permanent.Services))
	for _, s := range permanent.Services {
		prefix := "  "
		if _, ok := runtimeSet[s]; !ok {
			prefix = "- "
		}
		right = append(right, prefix+s)
	}

	if len(left) == 0 {
		left = []string{dimStyle.Render("(none)")}
	}
	if len(right) == 0 {
		right = []string{dimStyle.Render("(none)")}
	}

	return left, right
}

func diffPorts(runtime, permanent *firewalld.Zone) ([]string, []string) {
	if runtime == nil || permanent == nil {
		return []string{dimStyle.Render("(loading)")}, []string{dimStyle.Render("(loading)")}
	}

	permanentExact := make(map[string]struct{}, len(permanent.Ports))
	permanentByPort := make(map[string]struct{}, len(permanent.Ports))
	for _, p := range permanent.Ports {
		key := p.Port + "/" + p.Protocol
		permanentExact[key] = struct{}{}
		permanentByPort[p.Port] = struct{}{}
	}

	runtimeExact := make(map[string]struct{}, len(runtime.Ports))
	runtimeByPort := make(map[string]struct{}, len(runtime.Ports))
	for _, p := range runtime.Ports {
		key := p.Port + "/" + p.Protocol
		runtimeExact[key] = struct{}{}
		runtimeByPort[p.Port] = struct{}{}
	}

	left := make([]string, 0, len(runtime.Ports))
	for _, p := range runtime.Ports {
		key := p.Port + "/" + p.Protocol
		prefix := "  "
		if _, ok := permanentExact[key]; ok {
			prefix = "  "
		} else if _, ok := permanentByPort[p.Port]; ok {
			prefix = "~ "
		} else {
			prefix = "+ "
		}
		left = append(left, prefix+key)
	}

	right := make([]string, 0, len(permanent.Ports))
	for _, p := range permanent.Ports {
		key := p.Port + "/" + p.Protocol
		prefix := "  "
		if _, ok := runtimeExact[key]; ok {
			prefix = "  "
		} else if _, ok := runtimeByPort[p.Port]; ok {
			prefix = "~ "
		} else {
			prefix = "- "
		}
		right = append(right, prefix+key)
	}

	if len(left) == 0 {
		left = []string{dimStyle.Render("(none)")}
	}
	if len(right) == 0 {
		right = []string{dimStyle.Render("(none)")}
	}

	return left, right
}

func diffRichRules(runtime, permanent *firewalld.Zone) ([]string, []string) {
	if runtime == nil || permanent == nil {
		return []string{dimStyle.Render("(loading)")}, []string{dimStyle.Render("(loading)")}
	}

	permanentSet := make(map[string]struct{}, len(permanent.RichRules))
	for _, r := range permanent.RichRules {
		permanentSet[r] = struct{}{}
	}
	runtimeSet := make(map[string]struct{}, len(runtime.RichRules))
	for _, r := range runtime.RichRules {
		runtimeSet[r] = struct{}{}
	}

	left := make([]string, 0, len(runtime.RichRules))
	for _, r := range runtime.RichRules {
		prefix := "  "
		if _, ok := permanentSet[r]; !ok {
			prefix = "+ "
		}
		left = append(left, prefix+r)
	}

	right := make([]string, 0, len(permanent.RichRules))
	for _, r := range permanent.RichRules {
		prefix := "  "
		if _, ok := runtimeSet[r]; !ok {
			prefix = "- "
		}
		right = append(right, prefix+r)
	}

	if len(left) == 0 {
		left = []string{dimStyle.Render("(none)")}
	}
	if len(right) == 0 {
		right = []string{dimStyle.Render("(none)")}
	}

	return left, right
}

func diffNetwork(runtime, permanent *firewalld.Zone) ([]string, []string) {
	if runtime == nil || permanent == nil {
		return []string{dimStyle.Render("(loading)")}, []string{dimStyle.Render("(loading)")}
	}

	left := make([]string, 0)
	right := make([]string, 0)

	left = append(left, "Masquerade:")
	right = append(right, "Masquerade:")

	rMasq := "off"
	if runtime.Masquerade {
		rMasq = "on"
	}
	pMasq := "off"
	if permanent.Masquerade {
		pMasq = "on"
	}
	if rMasq != pMasq {
		left = append(left, "~ "+rMasq)
		right = append(right, "~ "+pMasq)
	} else {
		left = append(left, "  "+rMasq)
		right = append(right, "  "+pMasq)
	}

	left = append(left, "", "Interfaces:")
	right = append(right, "", "Interfaces:")
	permanentIfaces := make(map[string]struct{}, len(permanent.Interfaces))
	for _, i := range permanent.Interfaces {
		permanentIfaces[i] = struct{}{}
	}
	runtimeIfaces := make(map[string]struct{}, len(runtime.Interfaces))
	for _, i := range runtime.Interfaces {
		runtimeIfaces[i] = struct{}{}
	}
	for _, i := range runtime.Interfaces {
		prefix := "  "
		if _, ok := permanentIfaces[i]; !ok {
			prefix = "+ "
		}
		left = append(left, prefix+i)
	}
	for _, i := range permanent.Interfaces {
		prefix := "  "
		if _, ok := runtimeIfaces[i]; !ok {
			prefix = "- "
		}
		right = append(right, prefix+i)
	}
	if len(runtime.Interfaces) == 0 {
		left = append(left, dimStyle.Render("(none)"))
	}
	if len(permanent.Interfaces) == 0 {
		right = append(right, dimStyle.Render("(none)"))
	}

	left = append(left, "", "Sources:")
	right = append(right, "", "Sources:")
	permanentSources := make(map[string]struct{}, len(permanent.Sources))
	for _, s := range permanent.Sources {
		permanentSources[s] = struct{}{}
	}
	runtimeSources := make(map[string]struct{}, len(runtime.Sources))
	for _, s := range runtime.Sources {
		runtimeSources[s] = struct{}{}
	}
	for _, s := range runtime.Sources {
		prefix := "  "
		if _, ok := permanentSources[s]; !ok {
			prefix = "+ "
		}
		left = append(left, prefix+s)
	}
	for _, s := range permanent.Sources {
		prefix := "  "
		if _, ok := runtimeSources[s]; !ok {
			prefix = "- "
		}
		right = append(right, prefix+s)
	}
	if len(runtime.Sources) == 0 {
		left = append(left, dimStyle.Render("(none)"))
	}
	if len(permanent.Sources) == 0 {
		right = append(right, dimStyle.Render("(none)"))
	}

	return left, right
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
		line := highlightMatch(s, m.searchQuery)
		if !m.permanent && m.permanentData != nil {
			if _, ok := permanentSet[s]; !ok {
				line = line + " *"
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

	for i, p := range current.Ports {
		base := fmt.Sprintf("%s/%s", p.Port, p.Protocol)
		line := highlightMatch(base, m.searchQuery)
		prefix := "  "
		if !m.permanent && m.permanentData != nil {
			mark := portDiffMark(p, m.permanentData)
			if mark != "" {
				line = line + " " + mark
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

func renderRichRulesList(b *strings.Builder, m Model, current *firewalld.Zone) {
	if len(current.RichRules) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		return
	}

	permanentSet := make(map[string]struct{})
	if !m.permanent && m.permanentData != nil {
		for _, r := range m.permanentData.RichRules {
			permanentSet[r] = struct{}{}
		}
	}

	for i, r := range current.RichRules {
		prefix := "  "
		line := highlightMatch(r, m.searchQuery)
		if !m.permanent && m.permanentData != nil {
			if _, ok := permanentSet[r]; !ok {
				line = line + " *"
			}
		}
		if i == m.richIndex {
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

func renderNetworkView(b *strings.Builder, m Model, current *firewalld.Zone) {
	masq := "OFF"
	if current.Masquerade {
		masq = "ON"
	}
	b.WriteString("Masquerade: " + masq + "\n\n")

	b.WriteString("Interfaces:\n")
	if len(current.Interfaces) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, i := range current.Interfaces {
			line := highlightMatch(i, m.searchQuery)
			b.WriteString("  - " + line + "\n")
		}
	}

	b.WriteString("\nSources:\n")
	if len(current.Sources) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, s := range current.Sources {
			line := highlightMatch(s, m.searchQuery)
			b.WriteString("  - " + line + "\n")
		}
	}
}

func portDiffMark(p firewalld.Port, permanent *firewalld.Zone) string {
	if permanent == nil {
		return ""
	}

	portExists := false
	for _, pp := range permanent.Ports {
		if pp.Port == p.Port && pp.Protocol == p.Protocol {
			return ""
		}
		if pp.Port == p.Port {
			portExists = true
		}
	}
	if portExists {
		return "~"
	}
	return "*"
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
	case inputSearch:
		label = "Search: "
	}
	return inputStyle.Render(label) + m.input.View()
}

func renderStatus(m Model) string {
	mode := "Runtime"
	if m.permanent {
		mode = "Permanent"
	}
	legend := ""
	if m.splitView {
		legend = "Legend: + added  - removed  ~ modified"
	} else if !m.permanent {
		legend = "Legend: * runtime-only  ~ differs"
	}
	searchHint := "  /: search"
	if m.searchQuery != "" {
		searchHint = "  /: search  n/N: next"
	}
	status := fmt.Sprintf("Mode: %s | 1/2/3/4: tabs  S: split  a: add  d: delete  c: commit  u: revert  Tab: focus  j/k: move  P: toggle  r: refresh  q: quit%s", mode, searchHint)
	if legend != "" {
		status = status + "\n" + legend
	}
	return statusStyle.Render(status)
}

func highlightMatch(text, query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return text
	}
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	idx := strings.Index(lowerText, lowerQuery)
	if idx < 0 {
		return text
	}
	end := idx + len(lowerQuery)
	if end > len(text) {
		return text
	}
	return text[:idx] + matchStyle.Render(text[idx:end]) + text[end:]
}

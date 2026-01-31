package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/models"
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
	mode := "Runtime"
	if m.permanent {
		mode = "Permanent"
	}
	if m.splitView {
		b.WriteString("Selected: " + zone + " [Split]\n\n")
		b.WriteString(m.renderTabs())
		b.WriteString("\n\n")
		b.WriteString(m.renderSplit())
	} else {
		b.WriteString("Selected: " + zone + " [" + mode + "]\n\n")
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
	}

	return mainStyle.Render(b.String())
}

func (m Model) renderFooter() string {
	mode := "R"
	if m.permanent {
		mode = "P"
	}
	split := "Off"
	if m.splitView {
		split = "On"
	}
	return "[q] Quit  [tab] Switch Panel  [↑↓] Navigate  [h/l] Tabs  [1-5] Jump  [P] Mode(" + mode + ")  [S] Split(" + split + ")  [D] Debug  [r] Refresh"
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
		line := "  " + markerForService(m.runtimeData, m.permaData, service, m.permanent) + " " + service
		if i == m.selectedService && m.focus == focusMain {
			line = selectedStyle.Render("› " + markerForService(m.runtimeData, m.permaData, service, m.permanent) + " " + service)
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
		key := portKey(port)
		line := "  " + markerForPort(m.runtimeData, m.permaData, key, m.permanent) + " " + port.Protocol + " " + itoa(port.Number)
		if i == m.selectedPort && m.focus == focusMain {
			line = selectedStyle.Render("› " + markerForPort(m.runtimeData, m.permaData, key, m.permanent) + " " + port.Protocol + " " + itoa(port.Number))
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
		line := "  " + markerForRule(m.runtimeData, m.permaData, rule, m.permanent) + " " + rule
		if i == m.selectedRule && m.focus == focusMain {
			line = selectedStyle.Render("› " + markerForRule(m.runtimeData, m.permaData, rule, m.permanent) + " " + rule)
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

func (m Model) renderSplit() string {
	switch m.tab {
	case tabServices:
		return renderSplitList("Runtime", "Permanent",
			diffListStrings(m.runtimeData.Services, m.permaData.Services))
	case tabPorts:
		return renderSplitList("Runtime", "Permanent",
			diffListStrings(portsToKeys(m.runtimeData.Ports), portsToKeys(m.permaData.Ports)))
	case tabRules:
		return renderSplitList("Runtime", "Permanent",
			diffListStrings(m.runtimeData.RichRules, m.permaData.RichRules))
	case tabMasquerade:
		return renderSplitMasquerade(m.runtimeData, m.permaData)
	case tabInfo:
		return renderSplitInfo(m.runtimeData, m.permaData)
	default:
		return "Split view available for Services/Ports/Rich Rules only."
	}
}

type diffList struct {
	left  []string
	right []string
}

func diffListStrings(runtime, permanent []string) diffList {
	rt := make(map[string]struct{}, len(runtime))
	pm := make(map[string]struct{}, len(permanent))
	for _, v := range runtime {
		rt[v] = struct{}{}
	}
	for _, v := range permanent {
		pm[v] = struct{}{}
	}

	left := make([]string, 0, len(runtime))
	for _, v := range runtime {
		_, inPerm := pm[v]
		if inPerm {
			left = append(left, "~ "+v)
		} else {
			left = append(left, "+ "+v)
		}
	}
	right := make([]string, 0, len(permanent))
	for _, v := range permanent {
		_, inRun := rt[v]
		if inRun {
			right = append(right, "~ "+v)
		} else {
			right = append(right, "- "+v)
		}
	}

	return diffList{left: left, right: right}
}

func renderSplitList(leftTitle, rightTitle string, lists diffList) string {
	left := append([]string{leftTitle}, lists.left...)
	right := append([]string{rightTitle}, lists.right...)
	leftBlock := strings.Join(left, "\n")
	rightBlock := strings.Join(right, "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(30).Render(leftBlock),
		lipgloss.NewStyle().Width(30).Render(rightBlock),
	)
}

func renderSplitMasquerade(runtime, permanent *models.ZoneData) string {
	left := []string{"Runtime"}
	right := []string{"Permanent"}
	if runtime != nil {
		left = append(left, "Masquerade: "+onOff(runtime.Masquerade))
		left = append(left, "")
		left = append(left, "Interfaces:")
		left = append(left, formatBulletList(runtime.Interfaces)...)
		left = append(left, "")
		left = append(left, "Sources:")
		left = append(left, formatBulletList(runtime.Sources)...)
	}
	if permanent != nil {
		right = append(right, "Masquerade: "+onOff(permanent.Masquerade))
		right = append(right, "")
		right = append(right, "Interfaces:")
		right = append(right, formatBulletList(permanent.Interfaces)...)
		right = append(right, "")
		right = append(right, "Sources:")
		right = append(right, formatBulletList(permanent.Sources)...)
	}
	leftBlock := strings.Join(left, "\n")
	rightBlock := strings.Join(right, "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(30).Render(leftBlock),
		lipgloss.NewStyle().Width(30).Render(rightBlock),
	)
}

func renderSplitInfo(runtime, permanent *models.ZoneData) string {
	left := []string{"Runtime"}
	right := []string{"Permanent"}
	if runtime != nil {
		left = append(left, "Zone: "+runtime.Zone)
		left = append(left, "Interfaces: "+strings.Join(runtime.Interfaces, ", "))
		left = append(left, "Sources: "+strings.Join(runtime.Sources, ", "))
	}
	if permanent != nil {
		right = append(right, "Zone: "+permanent.Zone)
		right = append(right, "Interfaces: "+strings.Join(permanent.Interfaces, ", "))
		right = append(right, "Sources: "+strings.Join(permanent.Sources, ", "))
	}
	leftBlock := strings.Join(left, "\n")
	rightBlock := strings.Join(right, "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(30).Render(leftBlock),
		lipgloss.NewStyle().Width(30).Render(rightBlock),
	)
}

func formatBulletList(items []string) []string {
	if len(items) == 0 {
		return []string{"  (none)"}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, "  • "+item)
	}
	return out
}

func onOff(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}

func markerForService(runtime, permanent *models.ZoneData, service string, viewingPermanent bool) string {
	inRun := containsString(runtime.Services, service)
	inPerm := containsString(permanent.Services, service)
	return pickMarker(inRun, inPerm, viewingPermanent)
}

func markerForRule(runtime, permanent *models.ZoneData, rule string, viewingPermanent bool) string {
	inRun := containsString(runtime.RichRules, rule)
	inPerm := containsString(permanent.RichRules, rule)
	return pickMarker(inRun, inPerm, viewingPermanent)
}

func markerForPort(runtime, permanent *models.ZoneData, portKey string, viewingPermanent bool) string {
	inRun := containsString(portsToKeys(runtime.Ports), portKey)
	inPerm := containsString(portsToKeys(permanent.Ports), portKey)
	return pickMarker(inRun, inPerm, viewingPermanent)
}

func pickMarker(inRun, inPerm, viewingPermanent bool) string {
	if inRun && inPerm {
		return "💾"
	}
	if viewingPermanent {
		if inPerm {
			return "💾"
		}
		return "⚡"
	}
	if inRun {
		return "⚡"
	}
	return "💾"
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func portsToKeys(ports []firewalld.Port) []string {
	keys := make([]string, 0, len(ports))
	for _, port := range ports {
		keys = append(keys, portKey(port))
	}
	return keys
}

func portKey(port firewalld.Port) string {
	return strconv.Itoa(port.Number) + "/" + port.Protocol
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

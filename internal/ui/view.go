//go:build linux
// +build linux

package ui

import (
	"fmt"
	"strings"

	"lazyfirewall/internal/backup"
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
	statusStyle      = lipgloss.NewStyle().Background(lipgloss.Color("#88c0d0")).Foreground(lipgloss.Color("#2e3440")).Padding(0, 1)
	statusTextStyle  = lipgloss.NewStyle().Background(lipgloss.Color("#88c0d0")).Foreground(lipgloss.Color("#3b4252"))
	statusMutedStyle = lipgloss.NewStyle().Background(lipgloss.Color("#88c0d0")).Foreground(lipgloss.Color("#3b4252"))
	statusKeyStyle   = lipgloss.NewStyle().Background(lipgloss.Color("#88c0d0")).Foreground(lipgloss.Color("#eceff4"))
	sidebarStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	mainStyle        = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	warnStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	panicStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("1")).Padding(0, 1).Bold(true)
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
		active := false
		if m.activeZones != nil {
			_, active = m.activeZones[zone]
		}
		name := zone
		if zone == m.defaultZone {
			name = name + " [D]"
		}
		if active {
			name = name + " [A]"
		}
		line := name
		if i == m.selected {
			prefix = "> "
			if m.focus == focusZones {
				line = selectedStyle.Render(name)
			} else {
				line = titleStyle.Render(name)
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
	if m.notice != "" {
		b.WriteString(dimStyle.Render(m.notice))
		b.WriteString("\n\n")
	}

	if m.readOnly {
		b.WriteString(warnStyle.Render("[RO] Read-Only Mode - Run with sudo for editing"))
		b.WriteString("\n\n")
	}
	if m.panicMode {
		b.WriteString(panicStyle.Render("[PANIC] MODE ACTIVE - ALL CONNECTIONS DROPPED"))
		b.WriteString("\n\n")
	}

	if m.helpMode {
		renderHelp(&b, m)
		return mainStyle.Width(width).Render(b.String())
	}
	if m.backupMode {
		renderBackupView(&b, m)
		return mainStyle.Width(width).Render(b.String())
	}

	current := m.currentData()
	if current == nil {
		if m.permanent && m.permanentDenied {
			b.WriteString(warnStyle.Render("No permission to read permanent config. Run with sudo."))
		} else if !m.permanent && m.runtimeDenied {
			b.WriteString(warnStyle.Render("No permission to read runtime settings. Run with sudo."))
		} else if !m.permanent && m.runtimeInvalid {
			b.WriteString(warnStyle.Render("Zone not present in runtime. Switch to Permanent or reload."))
		} else {
			b.WriteString(dimStyle.Render("No data loaded"))
		}
		return mainStyle.Width(width).Render(b.String())
	}

	b.WriteString("\n")
	b.WriteString(renderTabs(m))
	b.WriteString("\n\n")
	if m.splitView && m.tab != tabIPSets {
		b.WriteString(renderSplitView(m, width))
	} else {
		if m.logMode {
			renderLogsView(&b, m)
		} else if m.templateMode {
			renderTemplates(&b, m)
		} else if m.detailsMode && m.tab == tabServices {
			renderServiceDetails(&b, m)
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
			case tabIPSets:
				renderIPSetsView(&b, m)
			case tabInfo:
				renderInfoView(&b, m, current)
			}
		}
	}

	if m.inputMode != inputNone {
		if m.inputMode == inputPanicConfirm {
			b.WriteString(warnStyle.Render("This will DROP ALL network connections immediately."))
			b.WriteString("\n")
			if m.panicCountdown > 0 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("Type YES and wait %ds, then press Enter.", m.panicCountdown)))
			} else {
				b.WriteString(dimStyle.Render("Type YES and press Enter to confirm."))
			}
			b.WriteString("\n")
		}
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
	ipsetLabel := " IPSets "
	infoLabel := " Info "
	switch m.tab {
	case tabServices:
		serviceLabel = tabActiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		ipsetLabel = tabInactiveStyle.Render(ipsetLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabPorts:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabActiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		ipsetLabel = tabInactiveStyle.Render(ipsetLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabRich:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabActiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		ipsetLabel = tabInactiveStyle.Render(ipsetLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabNetwork:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabActiveStyle.Render(networkLabel)
		ipsetLabel = tabInactiveStyle.Render(ipsetLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabIPSets:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		ipsetLabel = tabActiveStyle.Render(ipsetLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabInfo:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		ipsetLabel = tabInactiveStyle.Render(ipsetLabel)
		infoLabel = tabActiveStyle.Render(infoLabel)
	}
	return serviceLabel + " " + portLabel + " " + richLabel + " " + networkLabel + " " + ipsetLabel + " " + infoLabel
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
	case tabIPSets:
		return []string{dimStyle.Render("(split view not available)")}, []string{dimStyle.Render("(split view not available)")}
	case tabInfo:
		return diffInfo(m.runtimeData, m.permanentData)
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

func diffInfo(runtime, permanent *firewalld.Zone) ([]string, []string) {
	if runtime == nil || permanent == nil {
		return []string{dimStyle.Render("(loading)")}, []string{dimStyle.Render("(loading)")}
	}

	left := make([]string, 0)
	right := make([]string, 0)

	left = append(left, "Target:")
	right = append(right, "Target:")
	rTarget := runtime.Target
	pTarget := permanent.Target
	if rTarget == "" {
		rTarget = "(none)"
	}
	if pTarget == "" {
		pTarget = "(none)"
	}
	if rTarget != pTarget {
		left = append(left, "~ "+rTarget)
		right = append(right, "~ "+pTarget)
	} else {
		left = append(left, "  "+rTarget)
		right = append(right, "  "+pTarget)
	}

	left = append(left, "", "ICMP Blocks:")
	right = append(right, "", "ICMP Blocks:")
	permanentSet := make(map[string]struct{}, len(permanent.IcmpBlocks))
	for _, r := range permanent.IcmpBlocks {
		permanentSet[r] = struct{}{}
	}
	runtimeSet := make(map[string]struct{}, len(runtime.IcmpBlocks))
	for _, r := range runtime.IcmpBlocks {
		runtimeSet[r] = struct{}{}
	}
	for _, r := range runtime.IcmpBlocks {
		prefix := "  "
		if _, ok := permanentSet[r]; !ok {
			prefix = "+ "
		}
		left = append(left, prefix+r)
	}
	for _, r := range permanent.IcmpBlocks {
		prefix := "  "
		if _, ok := runtimeSet[r]; !ok {
			prefix = "- "
		}
		right = append(right, prefix+r)
	}
	if len(runtime.IcmpBlocks) == 0 {
		left = append(left, dimStyle.Render("(none)"))
	}
	if len(permanent.IcmpBlocks) == 0 {
		right = append(right, dimStyle.Render("(none)"))
	}

	left = append(left, "", "ICMP Inversion:")
	right = append(right, "", "ICMP Inversion:")
	rInv := "off"
	if runtime.IcmpInvert {
		rInv = "on"
	}
	pInv := "off"
	if permanent.IcmpInvert {
		pInv = "on"
	}
	if rInv != pInv {
		left = append(left, "~ "+rInv)
		right = append(right, "~ "+pInv)
	} else {
		left = append(left, "  "+rInv)
		right = append(right, "  "+pInv)
	}

	left = append(left, "", "Short:")
	right = append(right, "", "Short:")
	rShort := runtime.Short
	pShort := permanent.Short
	if rShort == "" {
		rShort = "(none)"
	}
	if pShort == "" {
		pShort = "(none)"
	}
	if rShort != pShort {
		left = append(left, "~ "+rShort)
		right = append(right, "~ "+pShort)
	} else {
		left = append(left, "  "+rShort)
		right = append(right, "  "+pShort)
	}

	left = append(left, "", "Description:")
	right = append(right, "", "Description:")
	rDesc := runtime.Description
	pDesc := permanent.Description
	if rDesc == "" {
		rDesc = "(none)"
	}
	if pDesc == "" {
		pDesc = "(none)"
	}
	if rDesc != pDesc {
		left = append(left, "~ "+rDesc)
		right = append(right, "~ "+pDesc)
	} else {
		left = append(left, "  "+rDesc)
		right = append(right, "  "+pDesc)
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
			prefix = "> "
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
			prefix = "> "
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
			prefix = "> "
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
	b.WriteString("Masquerade: " + masq + " (m to toggle)\n\n")

	b.WriteString("Interfaces:\n")
	index := 0
	if len(current.Interfaces) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, i := range current.Interfaces {
			line := highlightMatch(i, m.searchQuery)
			prefix := "  "
			if index == m.networkIndex {
				prefix = "> "
				if m.focus == focusMain {
					line = selectedStyle.Render(line)
				} else {
					line = titleStyle.Render(line)
				}
			}
			b.WriteString(prefix + line + "\n")
			index++
		}
	}

	b.WriteString("\nSources:\n")
	if len(current.Sources) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, s := range current.Sources {
			line := highlightMatch(s, m.searchQuery)
			prefix := "  "
			if index == m.networkIndex {
				prefix = "> "
				if m.focus == focusMain {
					line = selectedStyle.Render(line)
				} else {
					line = titleStyle.Render(line)
				}
			}
			b.WriteString(prefix + line + "\n")
			index++
		}
	}

	b.WriteString("\n")
}

func renderLogsView(b *strings.Builder, m Model) {
	zone := m.logZone
	if zone == "" {
		zone = "(none)"
	}
	b.WriteString(titleStyle.Render("Logs"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Filter: firewalld/iptables, zone " + zone + " (best effort)"))
	b.WriteString("\n\n")

	if m.logLoading {
		b.WriteString(dimStyle.Render("Starting log stream..."))
		return
	}
	if m.logErr != nil {
		b.WriteString(warnStyle.Render("Error: " + m.logErr.Error()))
		return
	}
	lines := m.getLogLines()
	if len(lines) == 0 {
		b.WriteString(dimStyle.Render("  (no log lines yet)"))
		return
	}
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
}

func renderIPSetsView(b *strings.Builder, m Model) {
	if m.ipsetDenied {
		b.WriteString(warnStyle.Render("No permission to read IPSets. Run with sudo."))
		return
	}
	if m.ipsetLoading {
		b.WriteString(dimStyle.Render("Loading IPSets..."))
		return
	}
	if m.ipsetErr != nil {
		b.WriteString(warnStyle.Render(fmt.Sprintf("Error: %v", m.ipsetErr)))
		b.WriteString("\n\n")
	}
	if len(m.ipsets) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		return
	}

	attached := attachedIPSets(m.currentData())
	b.WriteString("IPSets:\n")
	for i, name := range m.ipsets {
		prefix := "  "
		line := highlightMatch(name, m.searchQuery)
		if _, ok := attached[name]; ok {
			line = line + dimStyle.Render(" [Z]")
		}
		if i == m.ipsetIndex {
			prefix = "> "
			if m.focus == focusMain {
				line = selectedStyle.Render(line)
			} else {
				line = titleStyle.Render(line)
			}
		}
		b.WriteString(prefix + line + "\n")
	}

	b.WriteString("\nEntries")
	if m.ipsetEntryName != "" {
		b.WriteString(" (" + m.ipsetEntryName + ")")
	}
	b.WriteString(":\n")
	if m.ipsetEntriesLoading {
		b.WriteString(dimStyle.Render("  (loading)"))
		return
	}
	if m.ipsetEntriesErr != nil {
		b.WriteString(warnStyle.Render(fmt.Sprintf("  Error: %v", m.ipsetEntriesErr)))
		return
	}
	if len(m.ipsetEntries) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		return
	}
	for _, entry := range m.ipsetEntries {
		b.WriteString("  - " + entry + "\n")
	}
}

func attachedIPSets(zone *firewalld.Zone) map[string]struct{} {
	if zone == nil {
		return nil
	}
	out := make(map[string]struct{})
	for _, src := range zone.Sources {
		if strings.HasPrefix(src, "ipset:") {
			out[strings.TrimPrefix(src, "ipset:")] = struct{}{}
		}
	}
	return out
}

func renderInfoView(b *strings.Builder, m Model, current *firewalld.Zone) {
	target := current.Target
	if target == "" {
		target = "(none)"
	}
	short := current.Short
	if short == "" {
		short = "(none)"
	}
	desc := current.Description
	if desc == "" {
		desc = "(none)"
	}

	b.WriteString("Target: " + highlightMatch(target, m.searchQuery) + "\n")
	b.WriteString("ICMP Inversion: ")
	if current.IcmpInvert {
		b.WriteString("ON\n")
	} else {
		b.WriteString("OFF\n")
	}

	b.WriteString("\nICMP Blocks:\n")
	if len(current.IcmpBlocks) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, r := range current.IcmpBlocks {
			line := highlightMatch(r, m.searchQuery)
			b.WriteString("  - " + line + "\n")
		}
	}

	b.WriteString("\nShort:\n")
	b.WriteString("  " + highlightMatch(short, m.searchQuery) + "\n")

	b.WriteString("\nDescription:\n")
	b.WriteString("  " + highlightMatch(desc, m.searchQuery) + "\n")
}

func renderServiceDetails(b *strings.Builder, m Model) {
	name := m.detailsName
	if name == "" {
		name = "(none)"
	}

	header := "Service Details: " + name
	if m.detailsLoading {
		header = header + " " + m.spinner.View()
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n\n")

	if m.detailsErr != nil {
		b.WriteString(errorStyle.Render("Error: " + m.detailsErr.Error()))
		b.WriteString("\n")
		return
	}
	if m.detailsLoading {
		b.WriteString(dimStyle.Render("Loading..."))
		b.WriteString("\n")
		return
	}
	if m.details == nil {
		b.WriteString(dimStyle.Render("No details available"))
		b.WriteString("\n")
		return
	}

	info := m.details
	if info.Short != "" {
		b.WriteString("Short: " + info.Short + "\n")
	}
	if info.Description != "" {
		b.WriteString("\nDescription:\n  " + info.Description + "\n")
	}

	b.WriteString("\nPorts:\n")
	if len(info.Ports) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, p := range info.Ports {
			b.WriteString("  - " + p.Port + "/" + p.Protocol + "\n")
		}
	}

	b.WriteString("\nModules:\n")
	if len(info.Modules) == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		b.WriteString("\n")
	} else {
		for _, mod := range info.Modules {
			b.WriteString("  - " + mod + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Press Enter or Esc to close"))
}

func renderHelp(b *strings.Builder, m Model) {
	b.WriteString(titleStyle.Render("Help"))
	b.WriteString("\n\n")

	b.WriteString("Global:\n")
	b.WriteString("  ?           Toggle help\n")
	b.WriteString("  q / Ctrl+C  Quit\n\n")
	b.WriteString("  --dry-run/-n  Start in dry-run mode\n\n")
	b.WriteString("  --log-level   Set log level (debug|info|warn|error)\n")
	b.WriteString("  --no-color    Disable color output\n\n")
	b.WriteString("  Docs: README.md\n\n")

	b.WriteString("Navigation:\n")
	b.WriteString("  Tab         Switch focus\n")
	b.WriteString("  j/k         Move selection\n")
	b.WriteString("  1-6         Switch tabs\n")
	b.WriteString("  h/l         Prev/next tab\n\n")

	b.WriteString("View:\n")
	b.WriteString("  P           Toggle runtime/permanent\n")
	b.WriteString("  S           Split diff view\n")
	b.WriteString("  L           Toggle logs\n")
	b.WriteString("  r           Refresh data\n\n")

	b.WriteString("Actions:\n")
	b.WriteString("  n (zones)   New zone\n")
	b.WriteString("  d (zones)   Delete zone\n")
	b.WriteString("  D (zones)   Set default zone\n")
	b.WriteString("  a (main)    Add service/port\n")
	b.WriteString("  d (main)    Remove service/port\n")
	b.WriteString("  e           Edit rich rule\n")
	b.WriteString("  m           Toggle masquerade\n")
	b.WriteString("  i           Add interface\n")
	b.WriteString("  s           Add source\n")
	b.WriteString("  c           Commit runtime -> permanent\n")
	b.WriteString("  u           Reload (revert runtime)\n")
	b.WriteString("  t           Apply template\n")
	b.WriteString("  Alt+P       Panic mode (type YES)\n")
	b.WriteString("  Ctrl+R      Backup restore menu\n")
	b.WriteString("  Ctrl+B      Create backup\n")
	b.WriteString("  Ctrl+E      Export zone (JSON/XML)\n")
	b.WriteString("  Alt+I       Import zone (JSON/XML)\n")
	b.WriteString("  Ctrl+Z/Y    Undo / Redo\n")
	b.WriteString("  Tab         Autocomplete (export/import/service)\n")
	b.WriteString("  Enter       Service details\n\n")
	b.WriteString("  n (ipsets)  New IPSet (permanent)\n")
	b.WriteString("  a (ipsets)  Add entry\n")
	b.WriteString("  d (ipsets)  Remove entry\n\n")
	b.WriteString("  D (ipsets)  Delete IPSet\n\n")

	b.WriteString("Search:\n")
	b.WriteString("  /           Search current tab\n")
	b.WriteString("  n / N       Next/prev match\n\n")

	b.WriteString("Indicators:\n")
	b.WriteString("  [A]         Active zone\n")
	b.WriteString("  [D]         Default zone\n")
	b.WriteString("  *           Runtime-only item\n")
	b.WriteString("  ~           Modified item\n")
	b.WriteString("  + / -       Added/removed (split view)\n\n")
	b.WriteString("  [Z]         IPSet attached to zone\n\n")

	b.WriteString(dimStyle.Render("Press Esc or ? to close"))
}

func renderTemplates(b *strings.Builder, m Model) {
	b.WriteString(titleStyle.Render("Apply Template"))
	b.WriteString("\n\n")
	for i, tpl := range defaultTemplates {
		prefix := "  "
		line := tpl.Name
		if i == m.templateIndex {
			prefix = "> "
			line = selectedStyle.Render(line)
		}
		b.WriteString(prefix + line + "\n")
	}

	if m.templateIndex >= 0 && m.templateIndex < len(defaultTemplates) {
		desc := defaultTemplates[m.templateIndex].Description
		if desc != "" {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(desc))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Enter to apply, Esc to cancel"))
}

func renderBackupView(b *strings.Builder, m Model) {
	zone := "Unknown"
	if len(m.zones) > 0 && m.selected < len(m.zones) {
		zone = m.zones[m.selected]
	}
	backupDir := ""
	if dir, err := backup.Dir(); err == nil {
		backupDir = dir
	}
	b.WriteString(titleStyle.Render("Backups: " + zone))
	b.WriteString("\n\n")

	if m.backupErr != nil {
		b.WriteString(errorStyle.Render("Error: " + m.backupErr.Error()))
		b.WriteString("\n\n")
	}

	if len(m.backupItems) == 0 {
		b.WriteString(dimStyle.Render("No backups found"))
		if backupDir != "" {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("Dir: " + backupDir))
		}
		b.WriteString("\n\n")
	} else {
		for i, item := range m.backupItems {
			prefix := "  "
			line := item.Time.Format("2006-01-02 15:04:05") + "  " + formatBytes(item.Size)
			if item.Description != "" {
				line = line + "  " + item.Description
			}
			if i == m.backupIndex {
				prefix = "> "
				line = selectedStyle.Render(line)
			}
			b.WriteString(prefix + line + "\n")
		}
		b.WriteString("\n")
		if m.backupPreview != "" {
			b.WriteString(titleStyle.Render("Preview"))
			b.WriteString("\n")
			b.WriteString(m.backupPreview)
			b.WriteString("\n\n")
		} else {
			b.WriteString(dimStyle.Render("Preview not available"))
			b.WriteString("\n\n")
		}
	}

	if m.readOnly {
		b.WriteString(warnStyle.Render("Read-only mode: restore disabled"))
		b.WriteString("\n")
	}
	b.WriteString(dimStyle.Render("Enter: restore  Esc/Ctrl+R: close  j/k: move"))
}

func formatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	kb := float64(size) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.1f GB", gb)
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
	case inputAddRich:
		label = "Add rich rule (" + mode + "): "
	case inputEditRich:
		label = "Edit rich rule (" + mode + "): "
	case inputAddInterface:
		label = "Add interface (" + mode + "): "
	case inputAddSource:
		label = "Add source (" + mode + "): "
	case inputAddZone:
		label = "Add zone: "
	case inputDeleteZone:
		label = "Delete zone (type name): "
	case inputPanicConfirm:
		label = "PANIC confirm: "
	case inputExportZone:
		label = "Export path: "
	case inputImportZone:
		label = "Import path: "
	case inputSearch:
		label = "Search: "
	case inputAddIPSet:
		label = "Add IPSet: "
	case inputAddIPSetEntry:
		label = "Add IPSet entry (" + mode + "): "
	case inputRemoveIPSetEntry:
		label = "Remove IPSet entry (" + mode + "): "
	case inputDeleteIPSet:
		label = "Delete IPSet: "
	case inputManualBackup:
		label = "Backup description: "
	}
	return inputStyle.Render(label) + m.input.View()
}

func renderStatus(m Model) string {
	mode := "Runtime"
	shortMode := "Run"
	if m.permanent {
		mode = "Permanent"
		shortMode = "Perm"
	}

	contextHints := []statusHint{
		{key: "a", label: "add"},
		{key: "d", label: "delete"},
		{key: "c", label: "commit"},
		{key: "u", label: "revert"},
	}
	if m.focus == focusZones {
		contextHints = []statusHint{
			{key: "n", label: "new zone"},
			{key: "d", label: "delete zone"},
			{key: "D", label: "default"},
		}
	} else if m.tab == tabIPSets {
		contextHints = []statusHint{
			{key: "n", label: "new ipset"},
			{key: "a", label: "add entry"},
			{key: "d", label: "remove entry"},
			{key: "D", label: "delete ipset"},
		}
	}

	rightHints := []statusHint{
		{key: "/", label: "search"},
		{key: "?", label: "help"},
		{key: "q", label: "quit"},
	}
	if m.searchQuery != "" {
		rightHints = append([]statusHint{{key: "n/N", label: "next"}}, rightHints...)
	}

	badges := []string{}
	if m.panicMode {
		badges = append(badges, statusKeyStyle.Render("[PANIC]"))
	}
	if m.dryRun {
		badges = append(badges, statusKeyStyle.Render("[DRY]"))
	}
	if m.readOnly {
		badges = append(badges, statusKeyStyle.Render("[RO]"))
	}

	contextCount := len(contextHints)
	includeTemplate := true
	includeMode := true
	modeIsFull := true

	buildLeft := func() string {
		parts := []string{}
		if len(badges) > 0 {
			parts = append(parts, strings.Join(badges, statusTextStyle.Render(" ")))
		}
		if includeMode {
			label := "Mode: " + mode
			if !modeIsFull {
				label = "Mode: " + shortMode
			}
			parts = append(parts, statusTextStyle.Render(label))
		}
		if contextCount > 0 {
			parts = append(parts, renderStatusHints(contextHints[:contextCount], false))
		}
		if includeTemplate {
			if m.readOnly {
				parts = append(parts, statusMutedStyle.Render("t: templates [RO]"))
			} else {
				parts = append(parts, renderStatusHints([]statusHint{{key: "t", label: "templates"}}, false))
			}
		}
		return joinStatusSegments(parts)
	}

	buildRight := func() string {
		return renderStatusHints(rightHints, false)
	}

	contentWidth := m.width - statusStyle.GetHorizontalFrameSize()
	firstLine := ""
	if contentWidth <= 0 {
		left := buildLeft()
		right := buildRight()
		if left != "" {
			firstLine = left + statusTextStyle.Render("  ") + right
		} else {
			firstLine = right
		}
	} else {
		for {
			left := buildLeft()
			right := buildRight()
			leftW := lipgloss.Width(left)
			rightW := lipgloss.Width(right)

			if left == "" {
				firstLine = right
				break
			}
			if leftW+rightW+1 <= contentWidth {
				gap := statusTextStyle.Render(strings.Repeat(" ", contentWidth-leftW-rightW))
				firstLine = left + gap + right
				break
			}

			switch {
			case includeTemplate:
				includeTemplate = false
			case contextCount > 0:
				contextCount--
			case modeIsFull:
				modeIsFull = false
			case includeMode:
				includeMode = false
			case len(badges) > 0:
				badges = badges[:len(badges)-1]
			case len(rightHints) > 1:
				rightHints = rightHints[:len(rightHints)-1]
			default:
				firstLine = right
				break
			}

			if firstLine != "" {
				break
			}
		}
	}

	legendParts := []string{}
	if m.tab != tabIPSets {
		if m.splitView {
			legendParts = append(legendParts, statusMutedStyle.Render("Legend: + added  - removed  ~ modified"))
		} else if !m.permanent {
			legendParts = append(legendParts, statusMutedStyle.Render("Legend: * runtime-only  ~ differs"))
		}
	}
	if m.tab == tabNetwork {
		legendParts = append(legendParts, renderStatusHints([]statusHint{
			{key: "i", label: "add interface"},
			{key: "s", label: "add source"},
			{key: "d", label: "remove selected"},
		}, false))
	}
	secondLine := " "
	if len(legendParts) > 0 {
		secondLine = strings.Join(legendParts, statusMutedStyle.Render(" | "))
	}

	targetWidth := m.width
	if targetWidth <= 0 {
		targetWidth = lipgloss.Width(firstLine)
		if w := lipgloss.Width(secondLine); w > targetWidth {
			targetWidth = w
		}
		targetWidth += statusStyle.GetHorizontalFrameSize()
	}

	minWidth := statusStyle.GetHorizontalFrameSize() + 1
	if targetWidth < minWidth {
		targetWidth = minWidth
	}

	return statusStyle.Copy().Width(targetWidth).Render(firstLine + "\n" + secondLine)
}

type statusHint struct {
	key   string
	label string
}

func renderStatusHints(hints []statusHint, muted bool) string {
	if len(hints) == 0 {
		return ""
	}
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		if muted {
			parts = append(parts, statusMutedStyle.Render(hint.key+": "+hint.label))
			continue
		}
		parts = append(parts, statusKeyStyle.Render(hint.key)+statusMutedStyle.Render(": "+hint.label))
	}
	return strings.Join(parts, statusTextStyle.Render("  "))
}

func joinStatusSegments(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, statusMutedStyle.Render(" | "))
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

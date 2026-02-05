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
	activeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	statusStyle      = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250")).Padding(0, 1)
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
		dot := "  "
		if active {
			dot = activeStyle.Render("●") + " "
		}
		line := dot + name
		if i == m.selected {
			prefix = "› "
			if m.focus == focusZones {
				line = dot + selectedStyle.Render(name)
			} else {
				line = dot + titleStyle.Render(name)
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

	if m.readOnly {
		b.WriteString(warnStyle.Render("🔒 Read-Only Mode - Run with sudo for editing"))
		b.WriteString("\n\n")
	}
	if m.panicMode {
		b.WriteString(panicStyle.Render("⚠ PANIC MODE ACTIVE - ALL CONNECTIONS DROPPED"))
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
	if m.splitView {
		b.WriteString(renderSplitView(m, width))
	} else {
		if m.templateMode {
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
	infoLabel := " Info "
	switch m.tab {
	case tabServices:
		serviceLabel = tabActiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabPorts:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabActiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabRich:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabActiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabNetwork:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabActiveStyle.Render(networkLabel)
		infoLabel = tabInactiveStyle.Render(infoLabel)
	case tabInfo:
		serviceLabel = tabInactiveStyle.Render(serviceLabel)
		portLabel = tabInactiveStyle.Render(portLabel)
		richLabel = tabInactiveStyle.Render(richLabel)
		networkLabel = tabInactiveStyle.Render(networkLabel)
		infoLabel = tabActiveStyle.Render(infoLabel)
	}
	return serviceLabel + " " + portLabel + " " + richLabel + " " + networkLabel + " " + infoLabel
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
				prefix = "› "
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
				prefix = "› "
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
	b.WriteString(dimStyle.Render("i: add interface  s: add source  d: remove selected"))
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

	b.WriteString("Navigation:\n")
	b.WriteString("  Tab         Switch focus\n")
	b.WriteString("  j/k         Move selection\n")
	b.WriteString("  1-5         Switch tabs\n")
	b.WriteString("  h/l         Prev/next tab\n\n")

	b.WriteString("View:\n")
	b.WriteString("  P           Toggle runtime/permanent\n")
	b.WriteString("  S           Split diff view\n")
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
	b.WriteString("  c           Commit runtime → permanent\n")
	b.WriteString("  u           Reload (revert runtime)\n")
	b.WriteString("  t           Apply template\n")
	b.WriteString("  Alt+P       Panic mode (type YES)\n")
	b.WriteString("  Ctrl+R      Backup restore menu\n")
	b.WriteString("  Enter       Service details\n\n")

	b.WriteString("Search:\n")
	b.WriteString("  /           Search current tab\n")
	b.WriteString("  n / N       Next/prev match\n\n")

	b.WriteString("Indicators:\n")
	b.WriteString("  ●           Active zone\n")
	b.WriteString("  *           Runtime-only item\n")
	b.WriteString("  ~           Modified item\n")
	b.WriteString("  + / -       Added/removed (split view)\n\n")

	b.WriteString(dimStyle.Render("Press Esc or ? to close"))
}

func renderTemplates(b *strings.Builder, m Model) {
	b.WriteString(titleStyle.Render("Apply Template"))
	b.WriteString("\n\n")
	for i, tpl := range defaultTemplates {
		prefix := "  "
		line := tpl.Name
		if i == m.templateIndex {
			prefix = "› "
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
			if i == m.backupIndex {
				prefix = "› "
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
	actions := "a: add  d: delete  c: commit  u: revert"
	if m.focus == focusZones {
		actions = "n: new zone  d: delete zone  D: default"
	}
	templates := "  t: templates"
	if m.readOnly {
		actions = dimStyle.Render(actions)
		templates = "  " + dimStyle.Render("t: templates 🔒")
	}
	prefix := ""
	if m.readOnly {
		prefix = "🔒 Read-Only | "
	}
	if m.panicMode {
		prefix = "🚨 PANIC | " + prefix
	}
	status := fmt.Sprintf("%sMode: %s | 1/2/3/4/5: tabs  S: split  %s  Tab: focus  j/k: move  P: toggle  r: refresh  ?: help  q: quit%s%s", prefix, mode, actions, searchHint, templates)
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

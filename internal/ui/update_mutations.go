//go:build linux
// +build linux

package ui

import (
	"fmt"

	"lazyfirewall/internal/firewalld"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) maybeBackup(zone string, needsBackup bool, cmd tea.Cmd) tea.Cmd {
	if !needsBackup {
		return cmd
	}
	if m.readOnly {
		return cmd
	}
	if zone == "" {
		return cmd
	}
	if m.backupDone == nil {
		m.backupDone = make(map[string]bool)
	}
	if m.backupDone[zone] {
		return cmd
	}
	m.pendingMutation = cmd
	return createBackupCmd(zone)
}

func (m *Model) setDryRunNotice(action string) {
	m.err = nil
	m.notice = "DRY RUN: would " + action
}

func modeLabel(permanent bool) string {
	if permanent {
		return "permanent"
	}
	return "runtime"
}

func (m *Model) pushUndo(action undoAction, clearRedo bool) {
	if len(m.undoStack) >= undoLimit {
		m.undoStack = m.undoStack[1:]
	}
	m.undoStack = append(m.undoStack, action)
	if clearRedo {
		m.redoStack = nil
	}
}

func (m *Model) pushRedo(action undoAction) {
	if len(m.redoStack) >= undoLimit {
		m.redoStack = m.redoStack[1:]
	}
	m.redoStack = append(m.redoStack, action)
}

func (m *Model) actionAddService(zone, service string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add service " + service, zone: zone}
	action.undo = removeServiceCmd(m.client, zone, service, permanent, action, recordRedo, false)
	action.redo = addServiceCmd(m.client, zone, service, permanent, action, recordUndo, false)
	return addServiceCmd(m.client, zone, service, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveService(zone, service string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove service " + service, zone: zone}
	action.undo = addServiceCmd(m.client, zone, service, permanent, action, recordRedo, false)
	action.redo = removeServiceCmd(m.client, zone, service, permanent, action, recordUndo, false)
	return removeServiceCmd(m.client, zone, service, permanent, action, recordUndo, true)
}

func (m *Model) actionAddPort(zone string, port firewalld.Port, permanent bool) tea.Cmd {
	label := port.Port + "/" + port.Protocol
	action := &undoAction{label: "add port " + label, zone: zone}
	action.undo = removePortCmd(m.client, zone, port, permanent, action, recordRedo, false)
	action.redo = addPortCmd(m.client, zone, port, permanent, action, recordUndo, false)
	return addPortCmd(m.client, zone, port, permanent, action, recordUndo, true)
}

func (m *Model) actionRemovePort(zone string, port firewalld.Port, permanent bool) tea.Cmd {
	label := port.Port + "/" + port.Protocol
	action := &undoAction{label: "remove port " + label, zone: zone}
	action.undo = addPortCmd(m.client, zone, port, permanent, action, recordRedo, false)
	action.redo = removePortCmd(m.client, zone, port, permanent, action, recordUndo, false)
	return removePortCmd(m.client, zone, port, permanent, action, recordUndo, true)
}

func (m *Model) actionAddRichRule(zone, rule string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add rich rule", zone: zone}
	action.undo = removeRichRuleCmd(m.client, zone, rule, permanent, action, recordRedo, false)
	action.redo = addRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, false)
	return addRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveRichRule(zone, rule string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove rich rule", zone: zone}
	action.undo = addRichRuleCmd(m.client, zone, rule, permanent, action, recordRedo, false)
	action.redo = removeRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, false)
	return removeRichRuleCmd(m.client, zone, rule, permanent, action, recordUndo, true)
}

func (m *Model) actionEditRichRule(zone, oldRule, newRule string, permanent bool) tea.Cmd {
	action := &undoAction{label: "edit rich rule", zone: zone}
	action.undo = updateRichRuleCmd(m.client, zone, newRule, oldRule, permanent, action, recordRedo, false)
	action.redo = updateRichRuleCmd(m.client, zone, oldRule, newRule, permanent, action, recordUndo, false)
	return updateRichRuleCmd(m.client, zone, oldRule, newRule, permanent, action, recordUndo, true)
}

func (m *Model) actionAddInterface(zone, iface string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add interface " + iface, zone: zone}
	action.undo = removeInterfaceCmd(m.client, zone, iface, permanent, action, recordRedo, false)
	action.redo = addInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, false)
	return addInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveInterface(zone, iface string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove interface " + iface, zone: zone}
	action.undo = addInterfaceCmd(m.client, zone, iface, permanent, action, recordRedo, false)
	action.redo = removeInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, false)
	return removeInterfaceCmd(m.client, zone, iface, permanent, action, recordUndo, true)
}

func (m *Model) actionAddSource(zone, source string, permanent bool) tea.Cmd {
	action := &undoAction{label: "add source " + source, zone: zone}
	action.undo = removeSourceCmd(m.client, zone, source, permanent, action, recordRedo, false)
	action.redo = addSourceCmd(m.client, zone, source, permanent, action, recordUndo, false)
	return addSourceCmd(m.client, zone, source, permanent, action, recordUndo, true)
}

func (m *Model) actionRemoveSource(zone, source string, permanent bool) tea.Cmd {
	action := &undoAction{label: "remove source " + source, zone: zone}
	action.undo = addSourceCmd(m.client, zone, source, permanent, action, recordRedo, false)
	action.redo = removeSourceCmd(m.client, zone, source, permanent, action, recordUndo, false)
	return removeSourceCmd(m.client, zone, source, permanent, action, recordUndo, true)
}

func (m *Model) actionMasquerade(zone string, enabled, permanent bool) tea.Cmd {
	state := "off"
	if enabled {
		state = "on"
	}
	action := &undoAction{label: "masquerade " + state, zone: zone}
	action.undo = setMasqueradeCmd(m.client, zone, !enabled, permanent, action, recordRedo, false)
	action.redo = setMasqueradeCmd(m.client, zone, enabled, permanent, action, recordUndo, false)
	return setMasqueradeCmd(m.client, zone, enabled, permanent, action, recordUndo, true)
}

func (m *Model) toggleMasquerade() tea.Cmd {
	current := m.currentData()
	if current == nil || len(m.zones) == 0 {
		return nil
	}
	zone := m.zones[m.selected]
	enabled := !current.Masquerade
	if m.dryRun {
		state := "on"
		if !enabled {
			state = "off"
		}
		m.setDryRunNotice(fmt.Sprintf("set masquerade %s for zone %s (%s)", state, zone, modeLabel(m.permanent)))
		return nil
	}
	return m.maybeBackup(zone, true, m.actionMasquerade(zone, enabled, m.permanent))
}

func (m *Model) removeSelected() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.tab == tabIPSets {
		return m.startRemoveIPSetEntry()
	}
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
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("remove service %s from zone %s (%s)", service, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionRemoveService(zone, service, m.permanent))
	case tabPorts:
		if len(current.Ports) == 0 {
			return nil
		}
		port := current.Ports[m.portIndex]
		if m.dryRun {
			label := port.Port + "/" + port.Protocol
			m.setDryRunNotice(fmt.Sprintf("remove port %s from zone %s (%s)", label, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionRemovePort(zone, port, m.permanent))
	case tabRich:
		if len(current.RichRules) == 0 {
			return nil
		}
		rule := current.RichRules[m.richIndex]
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("remove rich rule from zone %s (%s)", zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionRemoveRichRule(zone, rule, m.permanent))
	case tabNetwork:
		items := m.networkItems()
		if len(items) == 0 {
			return nil
		}
		if m.networkIndex < 0 || m.networkIndex >= len(items) {
			return nil
		}
		item := items[m.networkIndex]
		switch item.kind {
		case "iface":
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("remove interface %s from zone %s (%s)", item.value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionRemoveInterface(zone, item.value, m.permanent))
		case "source":
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("remove source %s from zone %s (%s)", item.value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionRemoveSource(zone, item.value, m.permanent))
		default:
			return nil
		}
	case tabInfo:
		m.err = fmt.Errorf("editing not implemented for this tab")
		return nil
	default:
		return nil
	}
}

func (m *Model) applyTemplate() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if len(m.zones) == 0 {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}
	current := m.currentData()
	if current == nil {
		m.err = fmt.Errorf("no data loaded")
		return nil
	}
	if m.templateIndex < 0 || m.templateIndex >= len(defaultTemplates) {
		m.err = fmt.Errorf("invalid template selection")
		return nil
	}

	tpl := defaultTemplates[m.templateIndex]
	services := filterMissingServices(tpl.Services, current.Services)
	ports := filterMissingPorts(tpl.Ports, current.Ports)
	if len(services) == 0 && len(ports) == 0 {
		m.err = fmt.Errorf("template already applied")
		return nil
	}

	m.templateMode = false
	if m.dryRun {
		zone := m.zones[m.selected]
		m.setDryRunNotice(fmt.Sprintf("apply template %s to zone %s (%s)", tpl.Name, zone, modeLabel(m.permanent)))
		return nil
	}
	m.loading = true
	m.err = nil
	zone := m.zones[m.selected]
	m.pendingZone = zone
	return m.maybeBackup(zone, true, applyTemplateCmd(m.client, zone, services, ports, m.permanent))
}

func filterMissingServices(template, current []string) []string {
	currentSet := make(map[string]struct{}, len(current))
	for _, s := range current {
		currentSet[s] = struct{}{}
	}
	out := make([]string, 0, len(template))
	for _, s := range template {
		if _, ok := currentSet[s]; !ok {
			out = append(out, s)
		}
	}
	return out
}

func filterMissingPorts(template, current []firewalld.Port) []firewalld.Port {
	currentSet := make(map[string]struct{}, len(current))
	for _, p := range current {
		currentSet[p.Port+"/"+p.Protocol] = struct{}{}
	}
	out := make([]firewalld.Port, 0, len(template))
	for _, p := range template {
		if _, ok := currentSet[p.Port+"/"+p.Protocol]; !ok {
			out = append(out, p)
		}
	}
	return out
}

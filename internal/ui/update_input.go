//go:build linux
// +build linux

package ui

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/validation"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) startAddInput() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	m.err = nil
	if m.tab == tabNetwork || m.tab == tabInfo {
		if m.tab == tabNetwork {
			m.err = fmt.Errorf("use i/s/m in Network tab")
		} else {
			m.err = fmt.Errorf("editing not implemented for this tab")
		}
		return nil
	}
	if m.tab == tabIPSets {
		if m.currentIPSetName() == "" {
			m.err = fmt.Errorf("no ipset selected (press n to create)")
			return nil
		}
	}
	m.input.SetValue("")
	m.editRichOld = ""
	switch m.tab {
	case tabServices:
		m.input.Placeholder = "service name"
		m.inputMode = inputAddService
	case tabPorts:
		m.input.Placeholder = "port/proto (e.g. 80/tcp)"
		m.inputMode = inputAddPort
	case tabRich:
		m.input.Placeholder = "rich rule"
		m.inputMode = inputAddRich
	case tabIPSets:
		m.input.Placeholder = "entry (IP/CIDR/etc)"
		m.inputMode = inputAddIPSetEntry
	}
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddZone() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "zone name"
	m.inputMode = inputAddZone
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddIPSet() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if !m.permanent {
		m.err = fmt.Errorf("ipset creation is permanent-only (press P)")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "ipset name [type]"
	m.inputMode = inputAddIPSet
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startManualBackup() tea.Cmd {
	if len(m.zones) == 0 || m.selected >= len(m.zones) {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "backup description (optional)"
	m.inputMode = inputManualBackup
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startDeleteIPSet() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if !m.permanent {
		m.err = fmt.Errorf("ipset deletion is permanent-only (press P)")
		return nil
	}
	name := m.currentIPSetName()
	if name == "" {
		m.err = fmt.Errorf("no ipset selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "type ipset name to delete"
	m.inputMode = inputDeleteIPSet
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startRemoveIPSetEntry() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.currentIPSetName() == "" {
		m.err = fmt.Errorf("no ipset selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "entry to remove"
	m.inputMode = inputRemoveIPSetEntry
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startDeleteZone() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if len(m.zones) == 0 || m.selected >= len(m.zones) {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "type zone name to delete"
	m.inputMode = inputDeleteZone
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddInterface() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.tab != tabNetwork {
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "interface name"
	m.inputMode = inputAddInterface
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startAddSource() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	if m.tab != tabNetwork {
		return nil
	}
	m.err = nil
	m.input.SetValue("")
	m.input.Placeholder = "source (IP/CIDR)"
	m.inputMode = inputAddSource
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) startEditRich() tea.Cmd {
	if m.readOnly {
		m.err = firewalld.ErrPermissionDenied
		return nil
	}
	current := m.currentData()
	if current == nil || len(current.RichRules) == 0 {
		return nil
	}
	if m.richIndex < 0 || m.richIndex >= len(current.RichRules) {
		return nil
	}
	m.err = nil
	m.editRichOld = current.RichRules[m.richIndex]
	m.input.Placeholder = "rich rule"
	m.input.SetValue(m.editRichOld)
	m.inputMode = inputEditRich
	m.input.CursorEnd()
	m.input.Focus()
	return nil
}

func (m *Model) submitInput() tea.Cmd {
	if m.inputMode == inputNone {
		return nil
	}
	value := strings.TrimSpace(m.input.Value())
	requiresValue := m.inputMode != inputManualBackup && m.inputMode != inputSearch
	if value == "" && requiresValue {
		m.err = fmt.Errorf("input cannot be empty")
		return nil
	}

	if m.inputMode == inputPanicConfirm {
		if m.panicCountdown > 0 {
			m.err = fmt.Errorf("wait %ds then press Enter", m.panicCountdown)
			return nil
		}
		if !strings.EqualFold(value, "YES") {
			m.err = fmt.Errorf("type YES to confirm")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		if m.dryRun {
			m.setDryRunNotice("enable panic mode")
			return nil
		}
		m.panicAutoArmed = true
		return enablePanicModeCmd(m.client)
	}

	if m.inputMode == inputExportZone {
		current := m.currentData()
		if current == nil {
			m.err = fmt.Errorf("no data loaded")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		return exportZoneCmd(value, current)
	}

	if m.inputMode == inputImportZone {
		if len(m.zones) == 0 || m.selected >= len(m.zones) {
			m.err = fmt.Errorf("no zone selected")
			return nil
		}
		zone := m.zones[m.selected]
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("import zone from %s into %s", value, zone))
			return nil
		}
		return m.maybeBackup(zone, true, importZoneCmd(m.client, zone, value))
	}

	if m.inputMode == inputAddZone {
		if err := validation.IsValidZoneName(value); err != nil {
			m.err = fmt.Errorf("invalid zone name: %w", err)
			return nil
		}
		for _, z := range m.zones {
			if z == value {
				m.err = fmt.Errorf("zone already exists")
				return nil
			}
		}
		m.inputMode = inputNone
		m.input.Blur()
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add zone %s", value))
			return nil
		}
		m.loading = true
		m.err = nil
		m.runtimeInvalid = false
		m.pendingZone = value
		return addZoneCmd(m.client, value)
	}

	if m.inputMode == inputDeleteZone {
		if len(m.zones) == 0 || m.selected >= len(m.zones) {
			m.err = fmt.Errorf("no zone selected")
			return nil
		}
		zone := m.zones[m.selected]
		if err := validation.IsValidZoneName(value); err != nil {
			m.err = fmt.Errorf("invalid zone name: %w", err)
			return nil
		}
		if value != zone {
			m.err = fmt.Errorf("type zone name to confirm deletion")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("delete zone %s", zone))
			return nil
		}
		m.loading = true
		m.err = nil
		m.runtimeInvalid = false
		m.pendingZone = ""
		return m.maybeBackup(zone, true, removeZoneCmd(m.client, zone))
	}

	if m.inputMode == inputManualBackup {
		if len(m.zones) == 0 || m.selected >= len(m.zones) {
			m.err = fmt.Errorf("no zone selected")
			return nil
		}
		zone := m.zones[m.selected]
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		return createManualBackupCmd(zone, value)
	}

	if m.inputMode == inputAddIPSet {
		name, ipsetType, err := parseIPSetInput(value)
		if err != nil {
			m.err = err
			return nil
		}
		if err := validation.IsValidZoneName(name); err != nil {
			m.err = fmt.Errorf("invalid ipset name: %w", err)
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add ipset %s (%s)", name, ipsetType))
			return nil
		}
		m.ipsetLoading = true
		return addIPSetCmd(m.client, name, ipsetType)
	}

	if m.inputMode == inputAddIPSetEntry {
		name := m.currentIPSetName()
		if name == "" {
			m.err = fmt.Errorf("no ipset selected")
			return nil
		}
		if strings.TrimSpace(value) == "" {
			m.err = fmt.Errorf("entry cannot be empty")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add ipset entry to %s (%s)", name, modeLabel(m.permanent)))
			return nil
		}
		m.ipsetLoading = true
		return addIPSetEntryCmd(m.client, name, strings.TrimSpace(value), m.permanent)
	}

	if m.inputMode == inputRemoveIPSetEntry {
		name := m.currentIPSetName()
		if name == "" {
			m.err = fmt.Errorf("no ipset selected")
			return nil
		}
		if strings.TrimSpace(value) == "" {
			m.err = fmt.Errorf("entry cannot be empty")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("remove ipset entry from %s (%s)", name, modeLabel(m.permanent)))
			return nil
		}
		m.ipsetLoading = true
		return removeIPSetEntryCmd(m.client, name, strings.TrimSpace(value), m.permanent)
	}

	if m.inputMode == inputDeleteIPSet {
		name := m.currentIPSetName()
		if name == "" {
			m.err = fmt.Errorf("no ipset selected")
			return nil
		}
		if strings.TrimSpace(value) != name {
			m.err = fmt.Errorf("type ipset name to confirm deletion")
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		m.err = nil
		m.notice = ""
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("delete ipset %s", name))
			return nil
		}
		m.ipsetLoading = true
		return removeIPSetCmd(m.client, name)
	}

	if m.currentData() == nil || len(m.zones) == 0 {
		m.err = fmt.Errorf("no zone selected")
		return nil
	}

	zone := m.zones[m.selected]
	switch m.tab {
	case tabServices:
		m.inputMode = inputNone
		m.input.Blur()
		if !m.servicesLoading && m.servicesErr == nil && len(m.availableServices) > 0 && !m.serviceExists(value) {
			m.err = fmt.Errorf("unknown service: %s", value)
			return nil
		}
		if m.dryRun {
			m.setDryRunNotice(fmt.Sprintf("add service %s to zone %s (%s)", value, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionAddService(zone, value, m.permanent))
	case tabPorts:
		port, err := parsePortInput(value)
		if err != nil {
			m.err = err
			return nil
		}
		m.inputMode = inputNone
		m.input.Blur()
		if m.dryRun {
			label := port.Port + "/" + port.Protocol
			m.setDryRunNotice(fmt.Sprintf("add port %s to zone %s (%s)", label, zone, modeLabel(m.permanent)))
			return nil
		}
		return m.maybeBackup(zone, true, m.actionAddPort(zone, port, m.permanent))
	case tabRich:
		switch m.inputMode {
		case inputAddRich:
			m.inputMode = inputNone
			m.input.Blur()
			if err := validateRichRule(value); err != nil {
				m.err = err
				return nil
			}
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("add rich rule to zone %s (%s)", zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionAddRichRule(zone, value, m.permanent))
		case inputEditRich:
			oldRule := m.editRichOld
			m.editRichOld = ""
			m.inputMode = inputNone
			m.input.Blur()
			if oldRule == value {
				return nil
			}
			if err := validateRichRule(value); err != nil {
				m.err = err
				return nil
			}
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("edit rich rule in zone %s (%s)", zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionEditRichRule(zone, oldRule, value, m.permanent))
		}
		return nil
	case tabNetwork:
		switch m.inputMode {
		case inputAddInterface:
			m.inputMode = inputNone
			m.input.Blur()
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("add interface %s to zone %s (%s)", value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionAddInterface(zone, value, m.permanent))
		case inputAddSource:
			if net.ParseIP(value) == nil {
				if _, _, err := net.ParseCIDR(value); err != nil {
					m.err = fmt.Errorf("invalid source: %s", value)
					return nil
				}
			}
			m.inputMode = inputNone
			m.input.Blur()
			if m.dryRun {
				m.setDryRunNotice(fmt.Sprintf("add source %s to zone %s (%s)", value, zone, modeLabel(m.permanent)))
				return nil
			}
			return m.maybeBackup(zone, true, m.actionAddSource(zone, value, m.permanent))
		}
		return nil
	default:
		return nil
	}
}

func (m *Model) completePath() {
	raw := m.input.Value()
	if raw == "" {
		m.notice = "Type a path, then press Tab"
		return
	}

	expanded, useTilde, home := expandUserPath(raw)
	dir, base := splitPath(expanded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		m.err = err
		return
	}

	type cand struct {
		name  string
		isDir bool
	}
	cands := make([]cand, 0)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, base) {
			cands = append(cands, cand{name: name, isDir: entry.IsDir()})
		}
	}
	if len(cands) == 0 {
		m.notice = "No matches"
		return
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].name < cands[j].name })
	names := make([]string, 0, len(cands))
	for _, c := range cands {
		if c.isDir {
			names = append(names, c.name+string(os.PathSeparator))
		} else {
			names = append(names, c.name)
		}
	}

	prefix := commonPrefix(names)
	if prefix == "" {
		prefix = base
	}

	newValue := joinPath(dir, prefix)
	if strings.HasSuffix(prefix, string(os.PathSeparator)) && !strings.HasSuffix(newValue, string(os.PathSeparator)) {
		newValue += string(os.PathSeparator)
	}
	if useTilde && home != "" && strings.HasPrefix(newValue, home) {
		newValue = "~" + strings.TrimPrefix(newValue, home)
	}

	m.input.SetValue(newValue)
	m.input.CursorEnd()

	if len(names) > 1 && prefix == base {
		m.notice = "Matches: " + strings.Join(limitList(names, 8), "  ")
	} else {
		m.notice = ""
	}
}

func (m *Model) serviceExists(name string) bool {
	if name == "" {
		return false
	}
	for _, s := range m.availableServices {
		if s == name {
			return true
		}
	}
	return false
}

func (m *Model) completeServiceName() {
	raw := strings.TrimSpace(m.input.Value())
	if raw == "" {
		m.notice = "Type a service name, then press Tab"
		return
	}
	if m.servicesLoading {
		m.notice = "Service list loading..."
		return
	}
	if m.servicesErr != nil {
		m.notice = "Service list unavailable"
		return
	}
	if len(m.availableServices) == 0 {
		m.notice = "No services available"
		return
	}
	matches := make([]string, 0)
	for _, name := range m.availableServices {
		if strings.HasPrefix(name, raw) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		m.notice = "No matches"
		return
	}
	sort.Strings(matches)
	prefix := commonPrefix(matches)
	if prefix == "" {
		prefix = raw
	}
	m.input.SetValue(prefix)
	m.input.CursorEnd()
	if len(matches) > 1 && prefix == raw {
		m.notice = "Matches: " + strings.Join(limitList(matches, 8), "  ")
	} else {
		m.notice = ""
	}
}

func expandUserPath(path string) (expanded string, useTilde bool, home string) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			useTilde = true
			home = h
			if path == "~" {
				return h, true, h
			}
			return filepath.Join(h, strings.TrimPrefix(path, "~/")), true, h
		}
	}
	return path, false, ""
}

func splitPath(path string) (dir string, base string) {
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		return path, ""
	}
	dir = filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	return dir, filepath.Base(path)
}

func joinPath(dir, base string) string {
	if dir == "." || dir == "" {
		return base
	}
	return filepath.Join(dir, base)
}

func commonPrefix(items []string) string {
	if len(items) == 0 {
		return ""
	}
	prefix := items[0]
	for _, item := range items[1:] {
		for !strings.HasPrefix(item, prefix) {
			if prefix == "" {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func limitList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	out := append([]string{}, items[:max]...)
	out = append(out, "…")
	return out
}

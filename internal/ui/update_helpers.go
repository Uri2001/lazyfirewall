//go:build linux
// +build linux

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lazyfirewall/internal/backup"
	"lazyfirewall/internal/firewalld"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) startZoneLoad(zone string, reset bool) tea.Cmd {
	m.loading = true
	m.pendingZone = zone
	m.ipsetLoading = true
	if m.logMode {
		m.logZone = zone
		m.clearLogLines()
		m.logErr = nil
	}
	if reset {
		m.detailsMode = false
		m.templateMode = false
		m.detailsName = ""
		m.details = nil
		m.detailsErr = nil
		m.detailsLoading = false
		m.runtimeDenied = false
		m.permanentDenied = false
		m.runtimeInvalid = false
		m.editRichOld = ""
		m.runtimeData = nil
		m.permanentData = nil
	}

	return tea.Batch(
		fetchZoneSettingsCmd(m.client, zone, false),
		fetchZoneSettingsCmd(m.client, zone, true),
		fetchIPSetsCmd(m.client, m.permanent),
	)
}

func (m *Model) toggleLogs() tea.Cmd {
	if m.logMode {
		m.logMode = false
		m.logLoading = false
		m.logErr = nil
		m.clearLogLines()
		m.logZone = ""
		if m.logCancel != nil {
			m.logCancel()
			m.logCancel = nil
		}
		m.logLineCh = nil
		return nil
	}
	m.logMode = true
	m.logLoading = true
	m.logErr = nil
	m.clearLogLines()
	m.logLineCh = nil
	m.logZone = ""
	if len(m.zones) > 0 && m.selected < len(m.zones) {
		m.logZone = m.zones[m.selected]
	}
	m.splitView = false
	m.templateMode = false
	m.detailsMode = false
	m.inputMode = inputNone
	m.input.Blur()
	return startLogStreamCmd()
}

const undoLimit = 20
const logLimit = 200

func (m *Model) currentData() *firewalld.Zone {
	if m.permanent {
		return m.permanentData
	}
	return m.runtimeData
}

func (m *Model) currentIPSetName() string {
	if len(m.ipsets) == 0 {
		return ""
	}
	if m.ipsetIndex < 0 || m.ipsetIndex >= len(m.ipsets) {
		return ""
	}
	return m.ipsets[m.ipsetIndex]
}

func (m *Model) fetchCurrentIPSetEntries() tea.Cmd {
	if m.tab != tabIPSets {
		return nil
	}
	name := m.currentIPSetName()
	if name == "" {
		return nil
	}
	m.ipsetEntriesLoading = true
	m.ipsetEntriesErr = nil
	m.ipsetEntryName = name
	return fetchIPSetEntriesCmd(m.client, name, m.permanent)
}

func (m *Model) currentService() string {
	current := m.currentData()
	if current == nil || len(current.Services) == 0 {
		return ""
	}
	if m.serviceIndex < 0 || m.serviceIndex >= len(current.Services) {
		return ""
	}
	return current.Services[m.serviceIndex]
}

func parsePortInput(value string) (firewalld.Port, error) {
	input := strings.TrimSpace(value)
	if input == "" {
		return firewalld.Port{}, fmt.Errorf("port input is empty")
	}

	var portStr string
	var proto string
	if strings.Contains(input, "/") {
		parts := strings.SplitN(input, "/", 2)
		portStr = strings.TrimSpace(parts[0])
		proto = strings.TrimSpace(parts[1])
	} else {
		fields := strings.Fields(input)
		if len(fields) != 2 {
			return firewalld.Port{}, fmt.Errorf("use format port/proto or \"port proto\"")
		}
		portStr = fields[0]
		proto = fields[1]
	}

	portNum, err := strconv.Atoi(portStr)
	if err != nil || portNum < 1 || portNum > 65535 {
		return firewalld.Port{}, fmt.Errorf("invalid port: %s", portStr)
	}

	proto = strings.ToLower(proto)
	switch proto {
	case "tcp", "udp", "sctp", "dccp":
	default:
		return firewalld.Port{}, fmt.Errorf("invalid protocol: %s", proto)
	}

	return firewalld.Port{Port: portStr, Protocol: proto}, nil
}

func parseIPSetInput(value string) (string, string, error) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return "", "", fmt.Errorf("ipset name is empty")
	}
	if len(fields) > 2 {
		return "", "", fmt.Errorf("use: name [type]")
	}
	name := fields[0]
	ipsetType := "hash:ip"
	if len(fields) == 2 {
		ipsetType = fields[1]
	}
	return name, ipsetType, nil
}

func validateRichRule(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("rich rule is empty")
	}
	if !strings.HasPrefix(trimmed, "rule") {
		return fmt.Errorf("rich rule must start with 'rule'")
	}
	return nil
}

func logMatchesZone(line, zone string) bool {
	if zone == "" {
		return true
	}
	lower := strings.ToLower(line)
	zoneLower := strings.ToLower(zone)
	if strings.Contains(lower, "zone=") || strings.Contains(lower, "zone:") {
		return strings.Contains(lower, "zone="+zoneLower) ||
			strings.Contains(lower, "zone:"+zoneLower) ||
			strings.Contains(lower, "zone: "+zoneLower)
	}
	return true
}

func logMatchesSource(line string) bool {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "firewalld") {
		return true
	}
	if strings.Contains(lower, "iptables") || strings.Contains(lower, "ip6tables") {
		return true
	}
	if strings.Contains(lower, "nftables") || strings.Contains(lower, " nft ") || strings.HasPrefix(lower, "nft") {
		return true
	}
	if (strings.Contains(lower, "in=") || strings.Contains(lower, "out=")) &&
		(strings.Contains(lower, "drop") || strings.Contains(lower, "reject") || strings.Contains(lower, "final_")) {
		return true
	}
	return false
}

func indexOfZone(zones []string, zone string) int {
	for i, z := range zones {
		if z == zone {
			return i
		}
	}
	return -1
}

func defaultExportPath(zone string) string {
	ts := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("zone-%s-%s.json", zone, ts)
	if dir, err := backup.Dir(); err == nil {
		base := filepath.Dir(dir)
		return filepath.Join(base, "exports", name)
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "lazyfirewall", "exports", name)
	}
	return name
}

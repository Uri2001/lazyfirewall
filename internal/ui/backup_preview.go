//go:build linux
// +build linux

package ui

import (
	"fmt"
	"strings"

	"lazyfirewall/internal/backup"
	"lazyfirewall/internal/firewalld"
)

func buildBackupPreview(path string, current *firewalld.Zone) (string, error) {
	backupZone, err := backup.ParseZoneXMLFile(path)
	if err != nil {
		return "", err
	}

	lines := []string{
		fmt.Sprintf("Backup contains: services %d, ports %d, interfaces %d, sources %d", len(backupZone.Services), len(backupZone.Ports), len(backupZone.Interfaces), len(backupZone.Sources)),
	}

	if current == nil {
		return strings.Join(lines, "\n"), nil
	}

	add, del := diffStringCounts(current.Services, backupZone.Services)
	lines = append(lines, fmt.Sprintf("Services: +%d  -%d", add, del))

	add, del = diffPortCounts(current.Ports, backupZone.Ports)
	lines = append(lines, fmt.Sprintf("Ports: +%d  -%d", add, del))

	add, del = diffStringCounts(current.Interfaces, backupZone.Interfaces)
	lines = append(lines, fmt.Sprintf("Interfaces: +%d  -%d", add, del))

	add, del = diffStringCounts(current.Sources, backupZone.Sources)
	lines = append(lines, fmt.Sprintf("Sources: +%d  -%d", add, del))

	if current.Masquerade != backupZone.Masquerade {
		lines = append(lines, fmt.Sprintf("Masquerade: %s → %s", onOff(current.Masquerade), onOff(backupZone.Masquerade)))
	}
	if current.Target != backupZone.Target && (current.Target != "" || backupZone.Target != "") {
		lines = append(lines, fmt.Sprintf("Target: %s → %s", emptyAsNone(current.Target), emptyAsNone(backupZone.Target)))
	}
	if current.Short != backupZone.Short && (current.Short != "" || backupZone.Short != "") {
		lines = append(lines, fmt.Sprintf("Short: %s → %s", emptyAsNone(current.Short), emptyAsNone(backupZone.Short)))
	}

	return strings.Join(lines, "\n"), nil
}

func diffStringCounts(current, backup []string) (add int, del int) {
	currentSet := make(map[string]struct{}, len(current))
	for _, s := range current {
		currentSet[s] = struct{}{}
	}
	backupSet := make(map[string]struct{}, len(backup))
	for _, s := range backup {
		backupSet[s] = struct{}{}
		if _, ok := currentSet[s]; !ok {
			add++
		}
	}
	for _, s := range current {
		if _, ok := backupSet[s]; !ok {
			del++
		}
	}
	return add, del
}

func diffPortCounts(current, backup []firewalld.Port) (add int, del int) {
	currentSet := make(map[string]struct{}, len(current))
	for _, p := range current {
		currentSet[p.Port+"/"+p.Protocol] = struct{}{}
	}
	backupSet := make(map[string]struct{}, len(backup))
	for _, p := range backup {
		key := p.Port + "/" + p.Protocol
		backupSet[key] = struct{}{}
		if _, ok := currentSet[key]; !ok {
			add++
		}
	}
	for key := range currentSet {
		if _, ok := backupSet[key]; !ok {
			del++
		}
	}
	return add, del
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func emptyAsNone(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

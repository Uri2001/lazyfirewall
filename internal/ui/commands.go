//go:build linux
// +build linux

package ui

import (
	"lazyfirewall/internal/firewalld"

	tea "github.com/charmbracelet/bubbletea"
)

type zonesMsg struct {
	zones []string
	err   error
}

type zoneSettingsMsg struct {
	zone     *firewalld.Zone
	zoneName string
	err      error
}

func fetchZonesCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		zones, err := client.ListZones()
		return zonesMsg{zones: zones, err: err}
	}
}

func fetchZoneSettingsCmd(client *firewalld.Client, zone string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		settings, err := client.GetZoneSettings(zone, permanent)
		return zoneSettingsMsg{zone: settings, zoneName: zone, err: err}
	}
}

package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"lazyfirewall/internal/firewalld"
)

func fetchZonesCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		zones, err := client.ListZones()
		return zonesMsg{zones: zones, err: err}
	}
}

func fetchActiveZonesCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		zones, err := client.GetActiveZones()
		return activeZonesMsg{zones: zones, err: err}
	}
}

func fetchDefaultZoneCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		zone, err := client.GetDefaultZone()
		return defaultZoneMsg{zone: zone, err: err}
	}
}

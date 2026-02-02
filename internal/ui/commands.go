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

type mutationMsg struct {
	zone string
	err  error
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

func addServiceCmd(client *firewalld.Client, zone, service string) tea.Cmd {
	return func() tea.Msg {
		err := client.AddServicePermanent(zone, service)
		return mutationMsg{zone: zone, err: err}
	}
}

func removeServiceCmd(client *firewalld.Client, zone, service string) tea.Cmd {
	return func() tea.Msg {
		err := client.RemoveServicePermanent(zone, service)
		return mutationMsg{zone: zone, err: err}
	}
}

func addPortCmd(client *firewalld.Client, zone string, port firewalld.Port) tea.Cmd {
	return func() tea.Msg {
		err := client.AddPortPermanent(zone, port)
		return mutationMsg{zone: zone, err: err}
	}
}

func removePortCmd(client *firewalld.Client, zone string, port firewalld.Port) tea.Cmd {
	return func() tea.Msg {
		err := client.RemovePortPermanent(zone, port)
		return mutationMsg{zone: zone, err: err}
	}
}

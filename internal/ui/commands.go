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

func addServiceCmd(client *firewalld.Client, zone, service string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.AddServicePermanent(zone, service)
		} else {
			err = client.AddServiceRuntime(zone, service)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func removeServiceCmd(client *firewalld.Client, zone, service string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.RemoveServicePermanent(zone, service)
		} else {
			err = client.RemoveServiceRuntime(zone, service)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func addPortCmd(client *firewalld.Client, zone string, port firewalld.Port, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.AddPortPermanent(zone, port)
		} else {
			err = client.AddPortRuntime(zone, port)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func removePortCmd(client *firewalld.Client, zone string, port firewalld.Port, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.RemovePortPermanent(zone, port)
		} else {
			err = client.RemovePortRuntime(zone, port)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func commitRuntimeCmd(client *firewalld.Client, zone string) tea.Cmd {
	return func() tea.Msg {
		err := client.RuntimeToPermanent()
		return mutationMsg{zone: zone, err: err}
	}
}

func reloadCmd(client *firewalld.Client, zone string) tea.Cmd {
	return func() tea.Msg {
		err := client.Reload()
		return mutationMsg{zone: zone, err: err}
	}
}

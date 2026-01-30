package ui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/models"
)

var errEmptyZone = errors.New("zone not selected")

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

func fetchZoneDataCmd(client *firewalld.Client, zone string) tea.Cmd {
	return func() tea.Msg {
		if zone == "" {
			return zoneDataMsg{err: errEmptyZone}
		}

		settings, err := client.GetZoneSettings(zone)
		if err != nil {
			return zoneDataMsg{err: err}
		}

		return zoneDataMsg{
			data: &models.ZoneData{
				Zone:       settings.Name,
				Services:   settings.Services,
				Ports:      settings.Ports,
				RichRules:  settings.RichRules,
				Masquerade: settings.Masquerade,
				Interfaces: settings.Interfaces,
				Sources:    settings.Sources,
			},
		}
	}
}

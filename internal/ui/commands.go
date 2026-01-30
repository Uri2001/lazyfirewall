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

		services, err := client.GetServices(zone, false)
		if err != nil {
			return zoneDataMsg{err: err}
		}
		ports, err := client.GetPorts(zone, false)
		if err != nil {
			return zoneDataMsg{err: err}
		}
		rules, err := client.GetRichRules(zone, false)
		if err != nil {
			return zoneDataMsg{err: err}
		}
		masq, err := client.GetMasqueradeStatus(zone, false)
		if err != nil {
			return zoneDataMsg{err: err}
		}
		ifaces, err := client.GetInterfaces(zone)
		if err != nil {
			return zoneDataMsg{err: err}
		}
		sources, err := client.GetSources(zone)
		if err != nil {
			return zoneDataMsg{err: err}
		}

		return zoneDataMsg{
			data: &models.ZoneData{
				Zone:       zone,
				Services:   services,
				Ports:      ports,
				RichRules:  rules,
				Masquerade: masq,
				Interfaces: ifaces,
				Sources:    sources,
			},
		}
	}
}

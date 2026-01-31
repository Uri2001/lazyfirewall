package ui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"

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

func fetchZoneDataCmd(client *firewalld.Client, zone string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		if zone == "" {
			return zoneDataMsg{err: errEmptyZone}
		}

		var (
			settings map[string]dbus.Variant
			err      error
		)
		if permanent {
			err = client.RawZoneSettingsPermanent(zone, &settings)
		} else {
			err = client.RawZoneSettings(zone, &settings)
		}
		if err != nil {
			return zoneDataMsg{err: err}
		}

		parsed, err := firewalld.ParseZoneSettings(zone, settings)
		if err != nil {
			return zoneDataMsg{err: err}
		}
		if !permanent && len(parsed.Ports) == 0 {
			if ports, err := client.GetPortsRuntime(zone); err == nil && len(ports) > 0 {
				parsed.Ports = ports
			}
		}

		rawKeys, rawPorts, rawDump := firewalld.DebugZoneSettings(settings)

		return zoneDataMsg{
			data: &models.ZoneData{
				Zone:       parsed.Name,
				Services:   parsed.Services,
				Ports:      parsed.Ports,
				RichRules:  parsed.RichRules,
				Masquerade: parsed.Masquerade,
				Interfaces: parsed.Interfaces,
				Sources:    parsed.Sources,
				RawKeys:    rawKeys,
				RawPorts:   rawPorts,
				RawDump:    rawDump,
			},
			perm: permanent,
		}
	}
}

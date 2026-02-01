package ui

import (
	"errors"
	"strconv"

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

func fetchServiceDetailsCmd(client *firewalld.Client, name string) tea.Cmd {
	return func() tea.Msg {
		info, err := client.GetServiceDetails(name)
		return serviceDetailsMsg{name: name, info: info, err: err}
	}
}

func addServiceCmd(client *firewalld.Client, zone, service string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		err := client.AddService(zone, service, permanent)
		if err != nil {
			return mutationMsg{err: err}
		}
		return mutationMsg{notice: "service added: " + service}
	}
}

func removeServiceCmd(client *firewalld.Client, zone, service string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		err := client.RemoveService(zone, service, permanent)
		if err != nil {
			return mutationMsg{err: err}
		}
		return mutationMsg{notice: "service removed: " + service}
	}
}

func addPortCmd(client *firewalld.Client, zone string, port firewalld.Port, permanent bool) tea.Cmd {
	return func() tea.Msg {
		err := client.AddPort(zone, port, permanent)
		if err != nil {
			return mutationMsg{err: err}
		}
		return mutationMsg{notice: "port added: " + port.Protocol + " " + strconv.Itoa(port.Number)}
	}
}

func removePortCmd(client *firewalld.Client, zone string, port firewalld.Port, permanent bool) tea.Cmd {
	return func() tea.Msg {
		err := client.RemovePort(zone, port, permanent)
		if err != nil {
			return mutationMsg{err: err}
		}
		return mutationMsg{notice: "port removed: " + port.Protocol + " " + strconv.Itoa(port.Number)}
	}
}

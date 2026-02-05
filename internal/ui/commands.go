//go:build linux
// +build linux

package ui

import (
	"time"

	"lazyfirewall/internal/backup"
	"lazyfirewall/internal/firewalld"

	tea "github.com/charmbracelet/bubbletea"
)

type zonesMsg struct {
	zones []string
	err   error
}

type activeZonesMsg struct {
	zones map[string][]string
	err   error
}

type signalsReadyMsg struct {
	ch     <-chan firewalld.SignalEvent
	cancel func()
	err    error
}

type signalsClosedMsg struct{}

type firewalldSignalMsg struct {
	event firewalld.SignalEvent
}

type panicModeMsg struct {
	enabled bool
	err     error
}

type panicToggleMsg struct {
	enabled bool
	err     error
}

type panicTickMsg struct{}

type panicAutoDisableMsg struct{}

type backupCreatedMsg struct {
	zone string
	err  error
}

type backupsMsg struct {
	zone  string
	items []backup.Backup
	err   error
}

type backupPreviewMsg struct {
	zone    string
	path    string
	preview string
	err     error
}

type backupRestoreMsg struct {
	zone string
	err  error
}

type zoneSettingsMsg struct {
	zone      *firewalld.Zone
	zoneName  string
	permanent bool
	err       error
}

type mutationMsg struct {
	zone string
	err  error
}

type defaultZoneMsg struct {
	zone string
	err  error
}

type serviceDetailsMsg struct {
	service string
	info    *firewalld.ServiceInfo
	err     error
}

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

func subscribeSignalsCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		ch, cancel, err := client.SubscribeSignals()
		return signalsReadyMsg{ch: ch, cancel: cancel, err: err}
	}
}

func listenSignalsCmd(ch <-chan firewalld.SignalEvent) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		ev, ok := <-ch
		if !ok {
			return signalsClosedMsg{}
		}
		return firewalldSignalMsg{event: ev}
	}
}

func fetchDefaultZoneCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		zone, err := client.GetDefaultZone()
		return defaultZoneMsg{zone: zone, err: err}
	}
}

func fetchPanicModeCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		enabled, err := client.QueryPanicMode()
		return panicModeMsg{enabled: enabled, err: err}
	}
}

func enablePanicModeCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.EnablePanicMode()
		return panicToggleMsg{enabled: true, err: err}
	}
}

func disablePanicModeCmd(client *firewalld.Client) tea.Cmd {
	return func() tea.Msg {
		err := client.DisablePanicMode()
		return panicToggleMsg{enabled: false, err: err}
	}
}

func panicTickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg {
		return panicTickMsg{}
	})
}

func panicAutoDisableCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return panicAutoDisableMsg{}
	})
}

func createBackupCmd(zone string) tea.Cmd {
	return func() tea.Msg {
		_, err := backup.CreateZoneBackup(zone)
		return backupCreatedMsg{zone: zone, err: err}
	}
}

func fetchBackupsCmd(zone string) tea.Cmd {
	return func() tea.Msg {
		items, err := backup.ListBackups(zone)
		return backupsMsg{zone: zone, items: items, err: err}
	}
}

func previewBackupCmd(zone, path string, current *firewalld.Zone) tea.Cmd {
	return func() tea.Msg {
		preview, err := buildBackupPreview(path, current)
		return backupPreviewMsg{zone: zone, path: path, preview: preview, err: err}
	}
}

func restoreBackupCmd(client *firewalld.Client, zone string, item backup.Backup) tea.Cmd {
	return func() tea.Msg {
		if err := backup.RestoreZoneBackup(zone, item); err != nil {
			return backupRestoreMsg{zone: zone, err: err}
		}
		if err := client.Reload(); err != nil {
			return backupRestoreMsg{zone: zone, err: err}
		}
		return backupRestoreMsg{zone: zone, err: nil}
	}
}

func fetchZoneSettingsCmd(client *firewalld.Client, zone string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		settings, err := client.GetZoneSettings(zone, permanent)
		return zoneSettingsMsg{zone: settings, zoneName: zone, permanent: permanent, err: err}
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

func addRichRuleCmd(client *firewalld.Client, zone, rule string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.AddRichRulePermanent(zone, rule)
		} else {
			err = client.AddRichRuleRuntime(zone, rule)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func removeRichRuleCmd(client *firewalld.Client, zone, rule string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.RemoveRichRulePermanent(zone, rule)
		} else {
			err = client.RemoveRichRuleRuntime(zone, rule)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func updateRichRuleCmd(client *firewalld.Client, zone, oldRule, newRule string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			if err = client.RemoveRichRulePermanent(zone, oldRule); err != nil {
				return mutationMsg{zone: zone, err: err}
			}
			err = client.AddRichRulePermanent(zone, newRule)
		} else {
			if err = client.RemoveRichRuleRuntime(zone, oldRule); err != nil {
				return mutationMsg{zone: zone, err: err}
			}
			err = client.AddRichRuleRuntime(zone, newRule)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func addInterfaceCmd(client *firewalld.Client, zone, iface string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.AddInterfacePermanent(zone, iface)
		} else {
			err = client.AddInterfaceRuntime(zone, iface)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func removeInterfaceCmd(client *firewalld.Client, zone, iface string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.RemoveInterfacePermanent(zone, iface)
		} else {
			err = client.RemoveInterfaceRuntime(zone, iface)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func addSourceCmd(client *firewalld.Client, zone, source string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.AddSourcePermanent(zone, source)
		} else {
			err = client.AddSourceRuntime(zone, source)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func removeSourceCmd(client *firewalld.Client, zone, source string, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			err = client.RemoveSourcePermanent(zone, source)
		} else {
			err = client.RemoveSourceRuntime(zone, source)
		}
		return mutationMsg{zone: zone, err: err}
	}
}

func setMasqueradeCmd(client *firewalld.Client, zone string, enabled, permanent bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if permanent {
			if enabled {
				err = client.EnableMasqueradePermanent(zone)
			} else {
				err = client.DisableMasqueradePermanent(zone)
			}
		} else {
			if enabled {
				err = client.EnableMasqueradeRuntime(zone)
			} else {
				err = client.DisableMasqueradeRuntime(zone)
			}
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

func addZoneCmd(client *firewalld.Client, zone string) tea.Cmd {
	return func() tea.Msg {
		if err := client.AddZonePermanent(zone); err != nil {
			return zonesMsg{err: err}
		}
		zones, err := client.ListZones()
		return zonesMsg{zones: zones, err: err}
	}
}

func removeZoneCmd(client *firewalld.Client, zone string) tea.Cmd {
	return func() tea.Msg {
		if err := client.RemoveZonePermanent(zone); err != nil {
			return zonesMsg{err: err}
		}
		zones, err := client.ListZones()
		return zonesMsg{zones: zones, err: err}
	}
}

func setDefaultZoneCmd(client *firewalld.Client, zone string) tea.Cmd {
	return func() tea.Msg {
		if err := client.SetDefaultZone(zone); err != nil {
			return defaultZoneMsg{err: err}
		}
		current, err := client.GetDefaultZone()
		if err != nil {
			return defaultZoneMsg{err: err}
		}
		return defaultZoneMsg{zone: current, err: nil}
	}
}

func fetchServiceDetailsCmd(client *firewalld.Client, service string) tea.Cmd {
	return func() tea.Msg {
		info, err := client.GetServiceDetails(service)
		return serviceDetailsMsg{service: service, info: info, err: err}
	}
}

func applyTemplateCmd(client *firewalld.Client, zone string, services []string, ports []firewalld.Port, permanent bool) tea.Cmd {
	return func() tea.Msg {
		if permanent {
			for _, s := range services {
				if err := client.AddServicePermanent(zone, s); err != nil {
					return mutationMsg{zone: zone, err: err}
				}
			}
			for _, p := range ports {
				if err := client.AddPortPermanent(zone, p); err != nil {
					return mutationMsg{zone: zone, err: err}
				}
			}
			return mutationMsg{zone: zone, err: nil}
		}

		for _, s := range services {
			if err := client.AddServiceRuntime(zone, s); err != nil {
				return mutationMsg{zone: zone, err: err}
			}
		}
		for _, p := range ports {
			if err := client.AddPortRuntime(zone, p); err != nil {
				return mutationMsg{zone: zone, err: err}
			}
		}
		return mutationMsg{zone: zone, err: nil}
	}
}

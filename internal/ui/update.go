//go:build linux
// +build linux

package ui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchZonesCmd(m.client),
		fetchDefaultZoneCmd(m.client),
		fetchActiveZonesCmd(m.client),
		fetchPanicModeCmd(m.client),
		fetchIPSetsCmd(m.client, m.permanent),
		fetchServiceCatalogCmd(m.client),
		subscribeSignalsCmd(m.client),
	)
}

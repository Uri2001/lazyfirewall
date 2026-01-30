package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"lazyfirewall/internal/firewalld"
)

type focusPanel int

const (
	focusSidebar focusPanel = iota
	focusMain
	focusDetails
)

type Model struct {
	client *firewalld.Client

	zones       []string
	activeZones map[string][]string
	defaultZone string

	selectedZone int
	focus        focusPanel

	width  int
	height int

	err error
}

func NewModel(client *firewalld.Client) Model {
	return Model{
		client: client,
		focus:  focusSidebar,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchZonesCmd(m.client), fetchActiveZonesCmd(m.client), fetchDefaultZoneCmd(m.client))
}

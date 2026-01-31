package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/models"
)

type focusPanel int

const (
	focusSidebar focusPanel = iota
	focusMain
	focusDetails
)

const focusCount = 2

type mainTab int

const (
	tabServices mainTab = iota
	tabPorts
	tabRules
	tabMasquerade
	tabInfo
)

type Model struct {
	client *firewalld.Client

	zones       []string
	activeZones map[string][]string
	defaultZone string
	zoneData    *models.ZoneData
	runtimeData *models.ZoneData
	permaData   *models.ZoneData
	loading     bool
	debugMode   bool
	permanent   bool
	splitView   bool

	serviceDetails    map[string]*firewalld.ServiceInfo
	serviceDetailsErr map[string]error
	serviceLoading    map[string]bool

	tab mainTab

	selectedZone    int
	focus           focusPanel
	selectedService int
	selectedPort    int
	selectedRule    int

	width  int
	height int

	err error
}

func NewModel(client *firewalld.Client) Model {
	return Model{
		client:            client,
		focus:             focusSidebar,
		tab:               tabServices,
		serviceDetails:    map[string]*firewalld.ServiceInfo{},
		serviceDetailsErr: map[string]error{},
		serviceLoading:    map[string]bool{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchZonesCmd(m.client), fetchActiveZonesCmd(m.client), fetchDefaultZoneCmd(m.client))
}

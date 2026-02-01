package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
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

type inputMode int

const (
	inputNone inputMode = iota
	inputAddService
	inputAddPort
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
	showHelp    bool

	serviceDetails    map[string]*firewalld.ServiceInfo
	serviceDetailsErr map[string]error
	serviceLoading    map[string]bool

	tab mainTab

	inputMode inputMode
	textInput textinput.Model
	inputErr  string
	status    string

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
	ti := textinput.New()
	ti.CharLimit = 64
	ti.Prompt = ""
	return Model{
		client:            client,
		focus:             focusSidebar,
		tab:               tabServices,
		textInput:         ti,
		serviceDetails:    map[string]*firewalld.ServiceInfo{},
		serviceDetailsErr: map[string]error{},
		serviceLoading:    map[string]bool{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchZonesCmd(m.client), fetchActiveZonesCmd(m.client), fetchDefaultZoneCmd(m.client))
}

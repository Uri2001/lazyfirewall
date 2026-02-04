//go:build linux
// +build linux

package ui

import (
	"time"

	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
)

type focusArea int

const (
	focusZones focusArea = iota
	focusMain
)

type mainTab int

const (
	tabServices mainTab = iota
	tabPorts
	tabRich
	tabNetwork
	tabInfo
)

type inputMode int

const (
	inputNone inputMode = iota
	inputAddService
	inputAddPort
	inputAddRich
	inputEditRich
	inputAddInterface
	inputAddSource
	inputAddZone
	inputDeleteZone
	inputSearch
)

type networkItem struct {
	kind  string
	value string
}

type Model struct {
	client    *firewalld.Client
	zones     []string
	selected  int
	focus     focusArea
	permanent bool

	tab             mainTab
	serviceIndex    int
	portIndex       int
	richIndex       int
	networkIndex    int
	splitView       bool
	searchQuery     string
	templateMode    bool
	templateIndex   int
	helpMode        bool
	readOnly        bool
	runtimeDenied   bool
	permanentDenied bool
	runtimeInvalid  bool
	defaultZone     string
	activeZones     map[string][]string
	editRichOld     string
	signals        <-chan firewalld.SignalEvent
	signalsCancel  func()
	signalRefresh  bool
	cache          map[string]*zoneCache
	cacheTTL       time.Duration
	cacheTick      time.Duration

	detailsMode    bool
	detailsLoading bool
	detailsName    string
	details        *firewalld.ServiceInfo
	detailsErr     error

	runtimeData   *firewalld.Zone
	permanentData *firewalld.Zone
	loading       bool
	pendingZone   string
	err           error

	width     int
	height    int
	spinner   spinner.Model
	input     textinput.Model
	inputMode inputMode
}

type zoneCache struct {
	runtime     *firewalld.Zone
	runtimeAt   time.Time
	permanent   *firewalld.Zone
	permanentAt time.Time
}

func NewModel(client *firewalld.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 32
	ti.Prompt = ""

	return Model{
		client:    client,
		focus:     focusZones,
		tab:       tabServices,
		loading:   true,
		spinner:   sp,
		input:     ti,
		inputMode: inputNone,
		permanent: false,
		readOnly:  client.ReadOnly(),
		cache:     make(map[string]*zoneCache),
		cacheTTL:  30 * time.Second,
		cacheTick: 30 * time.Second,
	}
}

func (m *Model) networkItems() []networkItem {
	current := m.currentData()
	if current == nil {
		return nil
	}
	items := make([]networkItem, 0, len(current.Interfaces)+len(current.Sources))
	for _, iface := range current.Interfaces {
		items = append(items, networkItem{kind: "iface", value: iface})
	}
	for _, src := range current.Sources {
		items = append(items, networkItem{kind: "source", value: src})
	}
	return items
}

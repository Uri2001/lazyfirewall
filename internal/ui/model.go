//go:build linux
// +build linux

package ui

import (
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
	inputSearch
)

type Model struct {
	client    *firewalld.Client
	zones     []string
	selected  int
	focus     focusArea
	permanent bool

	tab           mainTab
	serviceIndex  int
	portIndex     int
	richIndex     int
	splitView     bool
	searchQuery   string
	templateMode  bool
	templateIndex int
	helpMode      bool
	readOnly      bool

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

func NewModel(client *firewalld.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line

	ti := textinput.New()
	ti.CharLimit = 64
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
	}
}

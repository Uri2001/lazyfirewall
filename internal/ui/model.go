//go:build linux
// +build linux

package ui

import (
	"time"

	"lazyfirewall/internal/backup"
	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	tabIPSets
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
	inputPanicConfirm
	inputExportZone
	inputImportZone
	inputSearch
	inputAddIPSet
	inputAddIPSetEntry
	inputRemoveIPSetEntry
	inputDeleteIPSet
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

	tab                 mainTab
	serviceIndex        int
	portIndex           int
	richIndex           int
	networkIndex        int
	splitView           bool
	searchQuery         string
	templateMode        bool
	templateIndex       int
	helpMode            bool
	readOnly            bool
	runtimeDenied       bool
	permanentDenied     bool
	runtimeInvalid      bool
	defaultZone         string
	activeZones         map[string][]string
	editRichOld         string
	signals             <-chan firewalld.SignalEvent
	signalsCancel       func()
	signalRefresh       bool
	panicMode           bool
	panicCountdown      int
	panicAutoDur        time.Duration
	panicAutoArmed      bool
	backupMode          bool
	backupItems         []backup.Backup
	backupIndex         int
	backupPreview       string
	backupErr           error
	backupDone          map[string]bool
	pendingMutation     tea.Cmd
	notice              string
	undoStack           []undoAction
	redoStack           []undoAction
	ipsets              []string
	ipsetIndex          int
	ipsetEntries        []string
	ipsetEntryName      string
	ipsetLoading        bool
	ipsetEntriesLoading bool
	ipsetErr            error
	ipsetEntriesErr     error
	ipsetDenied         bool

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
	ti.CharLimit = 256
	ti.Width = 32
	ti.Prompt = ""

	return Model{
		client:       client,
		focus:        focusZones,
		tab:          tabServices,
		loading:      true,
		spinner:      sp,
		input:        ti,
		inputMode:    inputNone,
		permanent:    false,
		readOnly:     client.ReadOnly(),
		panicAutoDur: 10 * time.Minute,
		backupDone:   make(map[string]bool),
		ipsetLoading: true,
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

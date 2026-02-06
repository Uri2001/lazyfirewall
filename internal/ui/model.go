//go:build linux
// +build linux

package ui

import (
	"sync"
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
	inputManualBackup
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
	dryRun    bool

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
	availableServices   []string
	servicesLoading     bool
	servicesErr         error
	logMode             bool
	logLoading          bool
	logLinesStore       *logLinesStore
	logErr              error
	logZone             string
	logCancel           func()
	logLineCh           <-chan string

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

type Options struct {
	DryRun           bool
	NoColor          bool
	DefaultPermanent bool
}

func NewModel(client *firewalld.Client, opts Options) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 32
	ti.Prompt = ""

	return Model{
		client:          client,
		focus:           focusZones,
		tab:             tabServices,
		loading:         true,
		spinner:         sp,
		input:           ti,
		inputMode:       inputNone,
		permanent:       opts.DefaultPermanent,
		readOnly:        client.ReadOnly(),
		dryRun:          opts.DryRun,
		panicAutoDur:    10 * time.Minute,
		backupDone:      make(map[string]bool),
		ipsetLoading:    true,
		servicesLoading: true,
		logLinesStore:   &logLinesStore{},
	}
}

type logLinesStore struct {
	mu    sync.RWMutex
	lines []string
}

func (m *Model) ensureLogLinesStore() *logLinesStore {
	if m.logLinesStore == nil {
		m.logLinesStore = &logLinesStore{}
	}
	return m.logLinesStore
}

func (m *Model) appendLogLine(line string) {
	store := m.ensureLogLinesStore()
	store.mu.Lock()
	store.lines = append(store.lines, line)
	if len(store.lines) > logLimit {
		store.lines = store.lines[len(store.lines)-logLimit:]
	}
	store.mu.Unlock()
}

func (m Model) getLogLines() []string {
	if m.logLinesStore == nil {
		return nil
	}
	m.logLinesStore.mu.RLock()
	defer m.logLinesStore.mu.RUnlock()
	lines := make([]string, len(m.logLinesStore.lines))
	copy(lines, m.logLinesStore.lines)
	return lines
}

func (m *Model) clearLogLines() {
	store := m.ensureLogLinesStore()
	store.mu.Lock()
	store.lines = nil
	store.mu.Unlock()
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

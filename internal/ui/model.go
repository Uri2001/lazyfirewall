//go:build linux
// +build linux

package ui

import (
	"lazyfirewall/internal/firewalld"

	"github.com/charmbracelet/bubbles/spinner"
)

type focusArea int

const (
	focusZones focusArea = iota
	focusMain
)

type Model struct {
	client    *firewalld.Client
	zones     []string
	selected  int
	focus     focusArea
	permanent bool

	zoneData *firewalld.Zone
	loading  bool
	pendingZone string
	err      error

	width   int
	height  int
	spinner spinner.Model
}

func NewModel(client *firewalld.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line

	return Model{
		client:   client,
		focus:    focusZones,
		loading:  true,
		spinner:  sp,
		permanent: false,
	}
}

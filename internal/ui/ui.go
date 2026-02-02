//go:build linux
// +build linux

package ui

import (
	"lazyfirewall/internal/firewalld"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(client *firewalld.Client) error {
	model := NewModel(client)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

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
	m, err := program.Run()
	if finalModel, ok := m.(Model); ok {
		if finalModel.signalsCancel != nil {
			finalModel.signalsCancel()
		}
	}
	return err
}

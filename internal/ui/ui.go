//go:build linux
// +build linux

package ui

import (
	"context"
	"lazyfirewall/internal/firewalld"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func RunWithContext(ctx context.Context, client *firewalld.Client, opts Options) error {
	if opts.NoColor {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	model := NewModel(client, opts)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	m, err := program.Run()
	if finalModel, ok := m.(Model); ok {
		if finalModel.logCancel != nil {
			finalModel.logCancel()
		}
		if finalModel.signalsCancel != nil {
			finalModel.signalsCancel()
		}
	}
	return err
}

func Run(client *firewalld.Client, opts Options) error {
	return RunWithContext(context.Background(), client, opts)
}

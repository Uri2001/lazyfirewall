//go:build linux

package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/ui"
)

func main() {
	client, err := firewalld.NewClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Make sure firewalld is running: sudo systemctl start firewalld")
		os.Exit(1)
	}
	defer client.Close()

	model := ui.NewModel(client)
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		log.Fatal(err)
	}
}

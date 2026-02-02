//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/logger"
	"lazyfirewall/internal/ui"
)

func main() {
	logger.Init()

	client, err := firewalld.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprintln(os.Stderr, "Make sure firewalld is running:")
		fmt.Fprintln(os.Stderr, "  sudo systemctl start firewalld")
		os.Exit(1)
	}
	defer client.Close()

	if err := ui.Run(client); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
		os.Exit(1)
	}
}

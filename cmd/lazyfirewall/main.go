//go:build linux
// +build linux

package main

import (
	"flag"
	"fmt"
	"os"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/logger"
	"lazyfirewall/internal/ui"
	"lazyfirewall/internal/version"
)

func main() {
	var dryRun bool
	var showVersion bool
	flag.BoolVar(&dryRun, "dry-run", false, "show changes without applying")
	flag.BoolVar(&dryRun, "n", false, "alias for --dry-run")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	logger.Init()

	if showVersion {
		fmt.Println(version.String())
		return
	}

	client, err := firewalld.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprintln(os.Stderr, "Make sure firewalld is running:")
		fmt.Fprintln(os.Stderr, "  sudo systemctl start firewalld")
		os.Exit(1)
	}
	defer client.Close()

	if err := ui.Run(client, ui.Options{DryRun: dryRun}); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
		os.Exit(1)
	}
}

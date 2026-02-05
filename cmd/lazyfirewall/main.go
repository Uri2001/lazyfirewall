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
	var logLevel string
	var noColor bool
	flag.BoolVar(&dryRun, "dry-run", false, "show changes without applying")
	flag.BoolVar(&dryRun, "n", false, "alias for --dry-run")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&showVersion, "v", false, "alias for --version")
	flag.StringVar(&logLevel, "log-level", "", "set log level (debug|info|warn|error)")
	flag.BoolVar(&noColor, "no-color", false, "disable color output")
	flag.Parse()

	if err := logger.Init(logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

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

	if err := ui.Run(client, ui.Options{DryRun: dryRun, NoColor: noColor}); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
		os.Exit(1)
	}
}

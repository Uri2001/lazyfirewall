//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/logger"
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

	fmt.Printf("Connected to firewalld %s (%s)\n\n", client.Version(), client.APIVersion())

	zones, err := client.ListZones()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list zones: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Available zones:")
	for _, zone := range zones {
		fmt.Printf("  • %s\n", zone)
	}
}

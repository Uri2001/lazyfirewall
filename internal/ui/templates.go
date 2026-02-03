//go:build linux
// +build linux

package ui

import "lazyfirewall/internal/firewalld"

type zoneTemplate struct {
	Name        string
	Description string
	Services    []string
	Ports       []firewalld.Port
}

var defaultTemplates = []zoneTemplate{
	{
		Name:        "Web Server",
		Description: "Adds http and https services",
		Services:    []string{"http", "https"},
	},
	{
		Name:        "Database Server",
		Description: "Adds postgresql and mysql services",
		Services:    []string{"postgresql", "mysql"},
	},
	{
		Name:        "SSH Only",
		Description: "Adds ssh service (does not remove others)",
		Services:    []string{"ssh"},
	},
	{
		Name:        "Workstation",
		Description: "Adds common desktop services",
		Services:    []string{"ssh", "mdns", "samba-client", "ipp-client", "dhcpv6-client"},
	},
}

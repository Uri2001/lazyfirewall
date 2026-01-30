package models

import "lazyfirewall/internal/firewalld"

type ZoneData struct {
	Zone       string
	Services   []string
	Ports      []firewalld.Port
	RichRules  []string
	Masquerade bool
	Interfaces []string
	Sources    []string
	RawKeys    []string
	RawPorts   []string
	RawDump    []string
}

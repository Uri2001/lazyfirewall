//go:build linux
// +build linux

package firewalld

import "errors"

type Port struct {
	Port     string
	Protocol string
}

type ServiceInfo struct {
	Name        string
	Short       string
	Description string
	Ports       []Port
	Modules     []string
}

type Zone struct {
	Name        string
	Services    []string
	Ports       []Port
	RichRules   []string
	Masquerade  bool
	Interfaces  []string
	Sources     []string
	Target      string
	IcmpBlocks  []string
	IcmpInvert  bool
	Short       string
	Description string
}

var (
	ErrNotRunning       = errors.New("firewalld service is not running")
	ErrPermissionDenied = errors.New("permission denied (try sudo)")
	ErrUnsupportedAPI   = errors.New("firewalld version not supported")
	ErrInvalidZone      = errors.New("zone does not exist")
)

//go:build linux

package firewalld

type Port struct {
	Number   int
	Protocol string
}

type ServiceInfo struct {
	Name        string
	Ports       []Port
	Modules     []string
	Description string
}

type Zone struct {
	Name       string
	Target     string
	Interfaces []string
	Sources    []string
	Services   []string
	Ports      []Port
	Masquerade bool
	RichRules  []string
}

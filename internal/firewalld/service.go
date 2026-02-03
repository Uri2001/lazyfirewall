//go:build linux
// +build linux

package firewalld

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const serviceDir = "/usr/lib/firewalld/services"

type serviceXML struct {
	XMLName     xml.Name        `xml:"service"`
	Short       string          `xml:"short"`
	Description string          `xml:"description"`
	Ports       []servicePort   `xml:"port"`
	Modules     []serviceModule `xml:"module"`
}

type servicePort struct {
	Port     string `xml:"port,attr"`
	Protocol string `xml:"protocol,attr"`
}

type serviceModule struct {
	Name string `xml:"name,attr"`
}

func (c *Client) GetServiceDetails(name string) (*ServiceInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("service name is empty")
	}
	path := filepath.Join(serviceDir, name+".xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read service %s: %w", name, err)
	}

	var svc serviceXML
	if err := xml.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("parse service %s: %w", name, err)
	}

	info := &ServiceInfo{
		Name:        name,
		Short:       svc.Short,
		Description: svc.Description,
	}
	for _, p := range svc.Ports {
		info.Ports = append(info.Ports, Port{Port: p.Port, Protocol: p.Protocol})
	}
	for _, m := range svc.Modules {
		if m.Name != "" {
			info.Modules = append(info.Modules, m.Name)
		}
	}

	slog.Debug("service details loaded", "service", name, "ports", len(info.Ports))
	return info, nil
}

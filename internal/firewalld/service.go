//go:build linux
// +build linux

package firewalld

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var serviceDirs = []string{
	"/etc/firewalld/services",
	"/usr/lib/firewalld/services",
}

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
	var data []byte
	var lastErr error
	for _, dir := range serviceDirs {
		path := filepath.Join(dir, name+".xml")
		b, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		data = b
		lastErr = nil
		break
	}
	if lastErr != nil {
		return nil, fmt.Errorf("read service %s: %w", name, lastErr)
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

func (c *Client) ListServiceNames() ([]string, error) {
	seen := make(map[string]struct{})
	for _, dir := range serviceDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read services dir %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".xml") {
				continue
			}
			base := strings.TrimSuffix(name, ".xml")
			if base != "" {
				seen[base] = struct{}{}
			}
		}
	}
	services := make([]string, 0, len(seen))
	for name := range seen {
		services = append(services, name)
	}
	sort.Strings(services)
	return services, nil
}

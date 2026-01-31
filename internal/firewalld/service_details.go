//go:build linux

package firewalld

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
		return nil, errors.New("service name is empty")
	}
	path := filepath.Join("/usr/lib/firewalld/services", name+".xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc serviceXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	info := &ServiceInfo{
		Name:        name,
		Description: strings.TrimSpace(doc.Description),
	}
	short := strings.TrimSpace(doc.Short)
	if short != "" {
		if info.Description != "" {
			info.Description = short + "\n\n" + info.Description
		} else {
			info.Description = short
		}
	}

	for _, p := range doc.Ports {
		info.Ports = append(info.Ports, Port{
			Number:   parsePortNumber(p.Port),
			Protocol: p.Protocol,
		})
	}
	for _, m := range doc.Modules {
		if m.Name != "" {
			info.Modules = append(info.Modules, m.Name)
		}
	}

	return info, nil
}

func parsePortNumber(value string) int {
	if value == "" {
		return 0
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return number
}

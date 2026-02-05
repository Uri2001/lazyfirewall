//go:build linux
// +build linux

package backup

import (
	"encoding/xml"
	"os"

	"lazyfirewall/internal/firewalld"
)

type zoneXML struct {
	XMLName            xml.Name     `xml:"zone"`
	Target             string       `xml:"target,attr"`
	Short              string       `xml:"short"`
	Description        string       `xml:"description"`
	Services           []serviceXML `xml:"service"`
	Ports              []portXML    `xml:"port"`
	Interfaces         []ifaceXML   `xml:"interface"`
	Sources            []sourceXML  `xml:"source"`
	IcmpBlocks         []icmpXML    `xml:"icmp-block"`
	IcmpBlockInversion *struct{}    `xml:"icmp-block-inversion"`
	Masquerade         *struct{}    `xml:"masquerade"`
}

type serviceXML struct {
	Name string `xml:"name,attr"`
}

type portXML struct {
	Port     string `xml:"port,attr"`
	Protocol string `xml:"protocol,attr"`
}

type ifaceXML struct {
	Name string `xml:"name,attr"`
}

type sourceXML struct {
	Address string `xml:"address,attr"`
	Mac     string `xml:"mac,attr"`
	IPSet   string `xml:"ipset,attr"`
}

type icmpXML struct {
	Name string `xml:"name,attr"`
}

func ParseZoneXMLFile(path string) (*firewalld.Zone, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseZoneXML(data)
}

func ParseZoneXML(data []byte) (*firewalld.Zone, error) {
	var zx zoneXML
	if err := xml.Unmarshal(data, &zx); err != nil {
		return nil, err
	}

	z := &firewalld.Zone{
		Target:      zx.Target,
		Short:       zx.Short,
		Description: zx.Description,
		Masquerade:  zx.Masquerade != nil,
		IcmpInvert:  zx.IcmpBlockInversion != nil,
	}

	for _, s := range zx.Services {
		if s.Name != "" {
			z.Services = append(z.Services, s.Name)
		}
	}
	for _, p := range zx.Ports {
		if p.Port != "" && p.Protocol != "" {
			z.Ports = append(z.Ports, firewalld.Port{Port: p.Port, Protocol: p.Protocol})
		}
	}
	for _, i := range zx.Interfaces {
		if i.Name != "" {
			z.Interfaces = append(z.Interfaces, i.Name)
		}
	}
	for _, s := range zx.Sources {
		switch {
		case s.Address != "":
			z.Sources = append(z.Sources, s.Address)
		case s.Mac != "":
			z.Sources = append(z.Sources, "mac:"+s.Mac)
		case s.IPSet != "":
			z.Sources = append(z.Sources, "ipset:"+s.IPSet)
		}
	}
	for _, i := range zx.IcmpBlocks {
		if i.Name != "" {
			z.IcmpBlocks = append(z.IcmpBlocks, i.Name)
		}
	}
	return z, nil
}

//go:build linux
// +build linux

package backup

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lazyfirewall/internal/firewalld"
	"lazyfirewall/internal/validation"
)

const (
	maxParsedXMLSize = 1 << 20  // 1 MiB
	maxXMLFileSize   = 10 << 20 // 10 MiB
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
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxXMLFileSize {
		return nil, fmt.Errorf("XML file too large: %d bytes (max %d)", info.Size(), maxXMLFileSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseZoneXML(data)
}

func ParseZoneXML(data []byte) (*firewalld.Zone, error) {
	if len(data) > maxParsedXMLSize {
		return nil, fmt.Errorf("XML payload too large: %d bytes (max %d)", len(data), maxParsedXMLSize)
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true

	var zx zoneXML
	if err := decoder.Decode(&zx); err != nil {
		return nil, fmt.Errorf("failed to parse zone XML: %w", err)
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

func MarshalZoneXML(z *firewalld.Zone) ([]byte, error) {
	if z == nil {
		return nil, fmt.Errorf("zone is nil")
	}
	zx := zoneXML{
		Target:      z.Target,
		Short:       z.Short,
		Description: z.Description,
	}
	if z.Masquerade {
		zx.Masquerade = &struct{}{}
	}
	if z.IcmpInvert {
		zx.IcmpBlockInversion = &struct{}{}
	}
	for _, s := range z.Services {
		if s != "" {
			zx.Services = append(zx.Services, serviceXML{Name: s})
		}
	}
	for _, p := range z.Ports {
		if p.Port != "" && p.Protocol != "" {
			zx.Ports = append(zx.Ports, portXML{Port: p.Port, Protocol: p.Protocol})
		}
	}
	for _, i := range z.Interfaces {
		if i != "" {
			zx.Interfaces = append(zx.Interfaces, ifaceXML{Name: i})
		}
	}
	for _, s := range z.Sources {
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "mac:") {
			zx.Sources = append(zx.Sources, sourceXML{Mac: strings.TrimPrefix(s, "mac:")})
			continue
		}
		if strings.HasPrefix(s, "ipset:") {
			zx.Sources = append(zx.Sources, sourceXML{IPSet: strings.TrimPrefix(s, "ipset:")})
			continue
		}
		zx.Sources = append(zx.Sources, sourceXML{Address: s})
	}
	for _, i := range z.IcmpBlocks {
		if i != "" {
			zx.IcmpBlocks = append(zx.IcmpBlocks, icmpXML{Name: i})
		}
	}
	data, err := xml.MarshalIndent(zx, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), data...), nil
}

func WriteZoneXMLFile(zone string, z *firewalld.Zone) (string, error) {
	if err := validation.IsValidZoneName(zone); err != nil {
		return "", fmt.Errorf("invalid zone name: %w", err)
	}
	data, err := MarshalZoneXML(z)
	if err != nil {
		return "", err
	}
	dir := zoneConfigDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, zone+".xml")

	tmp := dest + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 10)
	defer func() {
		_ = os.Remove(tmp)
	}()

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	return dest, nil
}

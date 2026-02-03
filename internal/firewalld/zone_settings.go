//go:build linux
// +build linux

package firewalld

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/godbus/dbus/v5"
)

func (c *Client) GetZoneSettings(zone string, permanent bool) (*Zone, error) {
	if c.apiVersion != APIv2 {
		return nil, ErrUnsupportedAPI
	}

	if permanent {
		return c.getZoneSettingsPermanent(zone)
	}

	return c.getZoneSettingsRuntime(zone)
}

func (c *Client) getZoneSettingsRuntime(zone string) (*Zone, error) {
	slog.Debug("fetching runtime zone settings", "zone", zone)

	var settings map[string]dbus.Variant
	method := dbusInterface + ".zone.getZoneSettings2"
	if err := c.call(method, &settings, zone); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}

	return parseZoneSettings(zone, settings)
}

func (c *Client) getZoneSettingsPermanent(zone string) (*Zone, error) {
	slog.Debug("fetching permanent zone settings", "zone", zone)

	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return nil, err
	}

	var settings map[string]dbus.Variant
	method := dbusInterface + ".config.zone.getSettings2"
	if err := c.callObject(obj, method, &settings); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}

	return parseZoneSettings(zone, settings)
}

func (c *Client) getConfigZoneObject(zone string) (dbus.BusObject, error) {
	var path dbus.ObjectPath
	method := dbusInterface + ".config.getZoneByName"
	configObj := c.conn.Object(dbusInterface, dbusConfigPath)
	if err := c.callObject(configObj, method, &path, zone); err != nil {
		return nil, err
	}

	return c.conn.Object(dbusInterface, path), nil
}

func parseZoneSettings(zone string, settings map[string]dbus.Variant) (*Zone, error) {
	z := &Zone{Name: zone}

	if v, ok := settings["services"]; ok {
		z.Services = variantToStringSlice(v)
	}

	if v, ok := settings["ports"]; ok {
		ports, err := variantToPorts(v)
		if err != nil {
			slog.Warn("failed to parse ports", "zone", zone, "error", err)
		} else {
			z.Ports = ports
		}
	}

	if v, ok := settings["masquerade"]; ok {
		if val, ok := v.Value().(bool); ok {
			z.Masquerade = val
		} else {
			slog.Warn("unexpected masquerade type", "type", fmt.Sprintf("%T", v.Value()))
		}
	}

	if v, ok := settings["rules_str"]; ok {
		z.RichRules = variantToStringSlice(v)
	}

	if v, ok := settings["interfaces"]; ok {
		z.Interfaces = variantToStringSlice(v)
	}

	if v, ok := settings["sources"]; ok {
		z.Sources = variantToStringSlice(v)
	}

	if v, ok := settings["target"]; ok {
		if val, ok := v.Value().(string); ok {
			z.Target = val
		} else {
			slog.Warn("unexpected target type", "type", fmt.Sprintf("%T", v.Value()))
		}
	}

	if v, ok := settings["icmp_blocks"]; ok {
		z.IcmpBlocks = variantToStringSlice(v)
	}

	if v, ok := settings["icmp_block_inversion"]; ok {
		if val, ok := v.Value().(bool); ok {
			z.IcmpInvert = val
		} else {
			slog.Warn("unexpected icmp_block_inversion type", "type", fmt.Sprintf("%T", v.Value()))
		}
	}

	if v, ok := settings["short"]; ok {
		if val, ok := v.Value().(string); ok {
			z.Short = val
		} else {
			slog.Warn("unexpected short type", "type", fmt.Sprintf("%T", v.Value()))
		}
	}

	if v, ok := settings["description"]; ok {
		if val, ok := v.Value().(string); ok {
			z.Description = val
		} else {
			slog.Warn("unexpected description type", "type", fmt.Sprintf("%T", v.Value()))
		}
	}

	slog.Debug("zone parsed", "zone", zone, "services", len(z.Services), "ports", len(z.Ports))
	return z, nil
}

func variantToStringSlice(v dbus.Variant) []string {
	switch val := v.Value().(type) {
	case []string:
		return val
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			} else {
				slog.Warn("unexpected item type in string slice", "type", fmt.Sprintf("%T", item))
			}
		}
		return out
	default:
		slog.Warn("unexpected variant type for string slice", "type", fmt.Sprintf("%T", val))
		return nil
	}
}

func variantToPorts(v dbus.Variant) ([]Port, error) {
	switch val := v.Value().(type) {
	case [][]string:
		return parsePortTuples(val)
	case []string:
		return parsePortStrings(val)
	case []interface{}:
		return parsePortInterfaces(val)
	case [][]interface{}:
		return parsePortInterfaceTuples(val)
	default:
		return nil, fmt.Errorf("unexpected port format: %T", val)
	}
}

func parsePortStrings(items []string) ([]Port, error) {
	ports := make([]Port, 0, len(items))
	for _, item := range items {
		parts := strings.Split(item, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port string: %q", item)
		}
		ports = append(ports, Port{Port: parts[0], Protocol: parts[1]})
	}
	return ports, nil
}

func parsePortTuples(items [][]string) ([]Port, error) {
	ports := make([]Port, 0, len(items))
	for _, item := range items {
		if len(item) != 2 {
			return nil, fmt.Errorf("invalid port tuple: %v", item)
		}
		ports = append(ports, Port{Port: item[0], Protocol: item[1]})
	}
	return ports, nil
}

func parsePortInterfaces(items []interface{}) ([]Port, error) {
	ports := make([]Port, 0, len(items))
	for _, item := range items {
		switch val := item.(type) {
		case []string:
			parsed, err := parsePortTuples([][]string{val})
			if err != nil {
				return nil, err
			}
			ports = append(ports, parsed...)
		case []interface{}:
			if len(val) != 2 {
				return nil, fmt.Errorf("invalid port tuple: %v", val)
			}
			p, ok1 := val[0].(string)
			proto, ok2 := val[1].(string)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("invalid port tuple types: %T %T", val[0], val[1])
			}
			ports = append(ports, Port{Port: p, Protocol: proto})
		case string:
			parsed, err := parsePortStrings([]string{val})
			if err != nil {
				return nil, err
			}
			ports = append(ports, parsed...)
		default:
			return nil, fmt.Errorf("unexpected port tuple type: %T", val)
		}
	}
	return ports, nil
}

func parsePortInterfaceTuples(items [][]interface{}) ([]Port, error) {
	ports := make([]Port, 0, len(items))
	for _, item := range items {
		if len(item) != 2 {
			return nil, fmt.Errorf("invalid port tuple: %v", item)
		}
		p, ok1 := item[0].(string)
		proto, ok2 := item[1].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("invalid port tuple types: %T %T", item[0], item[1])
		}
		ports = append(ports, Port{Port: p, Protocol: proto})
	}
	return ports, nil
}

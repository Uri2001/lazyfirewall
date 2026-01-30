//go:build linux

package firewalld

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	dbusInterface = "org.fedoraproject.FirewallD1"
	dbusPath      = "/org/fedoraproject/FirewallD1"
)

type Client struct {
	conn *dbus.Conn
	obj  dbus.BusObject
}

func NewClient() (*Client, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect system bus: %w", err)
	}

	obj := conn.Object(dbusInterface, dbusPath)

	var state string
	call := obj.Call("org.freedesktop.DBus.Properties.Get", 0, dbusInterface, "state")
	if call.Err != nil {
		return nil, fmt.Errorf("firewalld not running: %w", call.Err)
	}
	if err := call.Store(&state); err != nil {
		return nil, fmt.Errorf("read firewalld state: %w", err)
	}

	return &Client{conn: conn, obj: obj}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) ListZones() ([]string, error) {
	var zones []string
	if err := c.call(dbusInterface+".zone.getZones", &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (c *Client) GetActiveZones() (map[string][]string, error) {
	var zones map[string][]string
	if err := c.call(dbusInterface+".getActiveZones", &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (c *Client) GetDefaultZone() (string, error) {
	var zone string
	if err := c.call(dbusInterface+".getDefaultZone", &zone); err != nil {
		return "", err
	}
	return zone, nil
}

func (c *Client) SetDefaultZone(zone string) error {
	return c.call(dbusInterface+".setDefaultZone", nil, zone)
}

func (c *Client) GetServices(zone string, permanent bool) ([]string, error) {
	if permanent {
		return nil, errors.New("permanent zone services not implemented")
	}
	settings, err := c.GetZoneSettings(zone)
	if err != nil {
		return nil, err
	}
	return settings.Services, nil
}

func (c *Client) GetPorts(zone string, permanent bool) ([]Port, error) {
	if permanent {
		return nil, errors.New("permanent zone ports not implemented")
	}
	settings, err := c.GetZoneSettings(zone)
	if err != nil {
		return nil, err
	}
	return settings.Ports, nil
}

func (c *Client) GetRichRules(zone string, permanent bool) ([]string, error) {
	if permanent {
		return nil, errors.New("permanent zone rich rules not implemented")
	}
	settings, err := c.GetZoneSettings(zone)
	if err != nil {
		return nil, err
	}
	return settings.RichRules, nil
}

func (c *Client) GetMasqueradeStatus(zone string, permanent bool) (bool, error) {
	if permanent {
		return false, errors.New("permanent zone masquerade not implemented")
	}
	settings, err := c.GetZoneSettings(zone)
	if err != nil {
		return false, err
	}
	return settings.Masquerade, nil
}

func (c *Client) GetInterfaces(zone string) ([]string, error) {
	settings, err := c.GetZoneSettings(zone)
	if err != nil {
		return nil, err
	}
	return settings.Interfaces, nil
}

func (c *Client) GetSources(zone string) ([]string, error) {
	settings, err := c.GetZoneSettings(zone)
	if err != nil {
		return nil, err
	}
	return settings.Sources, nil
}

func (c *Client) call(method string, out any, args ...any) error {
	if c == nil || c.obj == nil {
		return errors.New("firewalld client not initialized")
	}
	call := c.obj.Call(method, 0, args...)
	if call.Err != nil {
		return fmt.Errorf("dbus call %s: %w", method, call.Err)
	}
	if out == nil {
		return nil
	}
	if err := call.Store(out); err != nil {
		return fmt.Errorf("dbus store %s: %w", method, err)
	}
	return nil
}

func parsePortString(value string) (Port, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return Port{}, fmt.Errorf("invalid port format %q", value)
	}
	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return Port{}, fmt.Errorf("invalid port number %q: %w", parts[0], err)
	}
	return Port{Number: number, Protocol: parts[1]}, nil
}

func (c *Client) GetZoneSettings(zone string) (*Zone, error) {
	var settings map[string]dbus.Variant
	if err := c.call(dbusInterface+".zone.getZoneSettings2", &settings, zone); err != nil {
		return nil, err
	}
	return parseZoneSettings(zone, settings)
}

func parseZoneSettings(zone string, settings map[string]dbus.Variant) (*Zone, error) {
	z := &Zone{Name: zone}

	if v, ok := settings["services"]; ok {
		z.Services = toStringSlice(v.Value())
	}
	if v, ok := settings["interfaces"]; ok {
		z.Interfaces = toStringSlice(v.Value())
	}
	if v, ok := settings["sources"]; ok {
		z.Sources = toStringSlice(v.Value())
	}
	if v, ok := settings["masquerade"]; ok {
		if val, ok := v.Value().(bool); ok {
			z.Masquerade = val
		}
	}
	if v, ok := settings["rules_str"]; ok {
		z.RichRules = toStringSlice(v.Value())
	}
	if v, ok := settings["ports"]; ok {
		ports, err := toPorts(v.Value())
		if err != nil {
			return nil, err
		}
		z.Ports = ports
	}

	return z, nil
}

func toStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func toPorts(value any) ([]Port, error) {
	switch v := value.(type) {
	case [][]string:
		return portsFromStringPairs(v)
	case []dbus.Struct:
		pairs := make([][]string, 0, len(v))
		for _, entry := range v {
			if len(entry.Fields) < 2 {
				continue
			}
			pair := []string{}
			if s, ok := entry.Fields[0].(string); ok {
				pair = append(pair, s)
			}
			if s, ok := entry.Fields[1].(string); ok {
				pair = append(pair, s)
			}
			if len(pair) == 2 {
				pairs = append(pairs, pair)
			}
		}
		return portsFromStringPairs(pairs)
	case []interface{}:
		pairs := make([][]string, 0, len(v))
		for _, entry := range v {
			switch item := entry.(type) {
			case []string:
				if len(item) >= 2 {
					pairs = append(pairs, item[:2])
				}
			case []interface{}:
				if len(item) >= 2 {
					a, aok := item[0].(string)
					b, bok := item[1].(string)
					if aok && bok {
						pairs = append(pairs, []string{a, b})
					}
				}
			case dbus.Struct:
				if len(item.Fields) >= 2 {
					a, aok := item.Fields[0].(string)
					b, bok := item.Fields[1].(string)
					if aok && bok {
						pairs = append(pairs, []string{a, b})
					}
				}
			}
		}
		return portsFromStringPairs(pairs)
	default:
		return nil, nil
	}
}

func portsFromStringPairs(pairs [][]string) ([]Port, error) {
	ports := make([]Port, 0, len(pairs))
	for _, pair := range pairs {
		if len(pair) < 2 {
			continue
		}
		number, err := strconv.Atoi(pair[0])
		if err != nil {
			return nil, fmt.Errorf("invalid port number %q: %w", pair[0], err)
		}
		ports = append(ports, Port{Number: number, Protocol: pair[1]})
	}
	return ports, nil
}

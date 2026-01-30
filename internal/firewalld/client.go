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
	var services []string
	if err := c.call(dbusInterface+".zone.getServices", &services, zone, permanent); err != nil {
		return nil, err
	}
	return services, nil
}

func (c *Client) GetPorts(zone string, permanent bool) ([]Port, error) {
	if c == nil || c.obj == nil {
		return nil, errors.New("firewalld client not initialized")
	}

	call := c.obj.Call(dbusInterface+".zone.getPorts", 0, zone, permanent)
	if call.Err != nil {
		return nil, fmt.Errorf("dbus call %s: %w", dbusInterface+".zone.getPorts", call.Err)
	}

	var stringPorts []string
	if err := call.Store(&stringPorts); err == nil {
		ports := make([]Port, 0, len(stringPorts))
		for _, entry := range stringPorts {
			port, err := parsePortString(entry)
			if err != nil {
				return nil, err
			}
			ports = append(ports, port)
		}
		return ports, nil
	}

	type portTuple struct {
		Port     string
		Protocol string
	}
	var tuplePorts []portTuple
	if err := call.Store(&tuplePorts); err != nil {
		return nil, fmt.Errorf("dbus store %s: %w", dbusInterface+".zone.getPorts", err)
	}

	ports := make([]Port, 0, len(tuplePorts))
	for _, entry := range tuplePorts {
		number, err := strconv.Atoi(entry.Port)
		if err != nil {
			return nil, fmt.Errorf("invalid port number %q: %w", entry.Port, err)
		}
		ports = append(ports, Port{Number: number, Protocol: entry.Protocol})
	}
	return ports, nil
}

func (c *Client) GetRichRules(zone string, permanent bool) ([]string, error) {
	var rules []string
	if err := c.call(dbusInterface+".zone.getRichRules", &rules, zone, permanent); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *Client) GetMasqueradeStatus(zone string, permanent bool) (bool, error) {
	var enabled bool
	if err := c.call(dbusInterface+".zone.queryMasquerade", &enabled, zone, permanent); err != nil {
		return false, err
	}
	return enabled, nil
}

func (c *Client) GetInterfaces(zone string) ([]string, error) {
	var interfaces []string
	if err := c.call(dbusInterface+".zone.getInterfaces", &interfaces, zone); err != nil {
		return nil, err
	}
	return interfaces, nil
}

func (c *Client) GetSources(zone string) ([]string, error) {
	var sources []string
	if err := c.call(dbusInterface+".zone.getSources", &sources, zone); err != nil {
		return nil, err
	}
	return sources, nil
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

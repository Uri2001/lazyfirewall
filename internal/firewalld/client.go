//go:build linux

package firewalld

import (
	"errors"
	"fmt"

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

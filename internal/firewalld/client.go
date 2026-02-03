//go:build linux
// +build linux

package firewalld

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	dbusInterface  = "org.fedoraproject.FirewallD1"
	dbusPath       = "/org/fedoraproject/FirewallD1"
	dbusConfigPath = "/org/fedoraproject/FirewallD1/config"
	dbusBusPath    = "/org/freedesktop/DBus"
	dbusBusName    = "org.freedesktop.DBus"
)

type Client struct {
	conn       *dbus.Conn
	obj        dbus.BusObject
	version    string
	apiVersion APIVersion
	readOnly   bool
}

func NewClient() (*Client, error) {
	slog.Debug("connecting to system bus")

	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect system bus: %w", err)
	}

	obj := conn.Object(dbusInterface, dbusPath)
	client := &Client{
		conn: conn,
		obj:  obj,
	}

	busObj := conn.Object(dbusBusName, dbusBusPath)
	var hasOwner bool
	if err := busObj.Call("org.freedesktop.DBus.NameHasOwner", 0, dbusInterface).Store(&hasOwner); err != nil {
		conn.Close()
		return nil, fmt.Errorf("check firewalld owner: %w", err)
	}
	if !hasOwner {
		conn.Close()
		return nil, ErrNotRunning
	}

	var stateVar dbus.Variant
	if err := client.call("org.freedesktop.DBus.Properties.Get", &stateVar, dbusInterface, "state"); err != nil {
		if isPermissionDenied(err) {
			slog.Warn("state read denied", "error", err)
		} else {
			conn.Close()
			return nil, fmt.Errorf("read firewalld state: %w", err)
		}
	} else {
		state, _ := stateVar.Value().(string)
		slog.Info("firewalld state", "state", state)
	}

	if err := client.detectVersion(); err != nil {
		slog.Warn("version detection failed", "error", err)
	}

	if err := client.detectPermissions(); err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Version() string {
	return c.version
}

func (c *Client) APIVersion() APIVersion {
	return c.apiVersion
}

func (c *Client) ReadOnly() bool {
	return c.readOnly
}

func (c *Client) detectPermissions() error {
	slog.Debug("checking permissions")

	method := dbusInterface + ".authorizeAll"
	if err := c.call(method, nil); err != nil {
		if isPermissionDenied(err) {
			c.readOnly = true
			slog.Warn("read-only mode enabled", "error", err)
			return nil
		}
		return err
	}

	c.readOnly = false
	return nil
}

func isPermissionDenied(err error) bool {
	var dbusErr *dbus.Error
	if errors.As(err, &dbusErr) {
		switch dbusErr.Name {
		case "org.freedesktop.DBus.Error.AccessDenied",
			"org.fedoraproject.FirewallD1.AccessDenied",
			"org.fedoraproject.FirewallD1.NotAuthorized",
			"org.fedoraproject.FirewallD1.Error.AccessDenied",
			"org.fedoraproject.FirewallD1.Error.NotAuthorized":
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "accessdenied") ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "not authorized") ||
		strings.Contains(msg, "notauthorized") {
		return true
	}

	return false
}

func (c *Client) call(method string, out any, args ...any) error {
	slog.Debug("dbus call", "method", method, "args", args)

	call := c.obj.Call(method, 0, args...)
	if call.Err != nil {
		slog.Error("dbus call failed", "method", method, "error", call.Err)
		return fmt.Errorf("dbus %s: %w", method, call.Err)
	}

	if out == nil {
		return nil
	}

	if err := call.Store(out); err != nil {
		slog.Error("dbus store failed", "method", method, "error", err)
		return fmt.Errorf("dbus store %s: %w", method, err)
	}

	return nil
}

func (c *Client) callObject(obj dbus.BusObject, method string, out any, args ...any) error {
	slog.Debug("dbus call", "method", method, "args", args)

	call := obj.Call(method, 0, args...)
	if call.Err != nil {
		slog.Error("dbus call failed", "method", method, "error", call.Err)
		return fmt.Errorf("dbus %s: %w", method, call.Err)
	}

	if out == nil {
		return nil
	}

	if err := call.Store(out); err != nil {
		slog.Error("dbus store failed", "method", method, "error", err)
		return fmt.Errorf("dbus store %s: %w", method, err)
	}

	return nil
}

func (c *Client) ListZones() ([]string, error) {
	var zones []string

	method := dbusInterface + ".zone.getZones"
	if c.apiVersion == APIv1 {
		method = dbusInterface + ".getZones"
	}

	if err := c.call(method, &zones); err != nil {
		return nil, err
	}

	slog.Debug("zones listed", "count", len(zones), "zones", zones)
	return zones, nil
}

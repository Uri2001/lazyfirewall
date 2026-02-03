//go:build linux
// +build linux

package firewalld

import (
	"fmt"
	"log/slog"

	"github.com/godbus/dbus/v5"
)

func (c *Client) listZonesRuntime() ([]string, error) {
	var zones []string

	method := dbusInterface + ".zone.getZones"
	if c.apiVersion == APIv1 {
		method = dbusInterface + ".getZones"
	}

	if err := c.call(method, &zones); err != nil {
		return nil, err
	}

	slog.Debug("zones listed (runtime)", "count", len(zones), "zones", zones)
	return zones, nil
}

func (c *Client) listZonesPermanent() ([]string, error) {
	if c.apiVersion != APIv2 {
		return nil, ErrUnsupportedAPI
	}

	var zones []string
	method := dbusInterface + ".config.getZoneNames"
	configObj := c.conn.Object(dbusInterface, dbusConfigPath)
	if err := c.callObject(configObj, method, &zones); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}

	slog.Debug("zones listed (permanent)", "count", len(zones), "zones", zones)
	return zones, nil
}

func (c *Client) GetDefaultZone() (string, error) {
	if c.apiVersion != APIv2 {
		return "", ErrUnsupportedAPI
	}

	var zone string
	method := dbusInterface + ".getDefaultZone"
	if err := c.call(method, &zone); err != nil {
		if isPermissionDenied(err) {
			return "", ErrPermissionDenied
		}
		return "", err
	}
	return zone, nil
}

func (c *Client) SetDefaultZone(zone string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("setting default zone", "zone", zone)
	method := dbusInterface + ".setDefaultZone"
	return c.call(method, nil, zone)
}

func (c *Client) AddZonePermanent(zone string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding zone (permanent)", "zone", zone)
	method := dbusInterface + ".config.addZone2"
	settings := map[string]dbus.Variant{}
	configObj := c.conn.Object(dbusInterface, dbusConfigPath)
	if err := c.callObject(configObj, method, nil, zone, settings); err != nil {
		if isPermissionDenied(err) {
			return ErrPermissionDenied
		}
		return fmt.Errorf("add zone %s: %w", zone, err)
	}
	return nil
}

func (c *Client) RemoveZonePermanent(zone string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing zone (permanent)", "zone", zone)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.remove"
	if err := c.callObject(obj, method, nil); err != nil {
		if isPermissionDenied(err) {
			return ErrPermissionDenied
		}
		return fmt.Errorf("remove zone %s: %w", zone, err)
	}
	return nil
}

func (c *Client) GetActiveZones() (map[string][]string, error) {
	if c.apiVersion != APIv2 {
		return nil, ErrUnsupportedAPI
	}

	var zones map[string][]string
	method := dbusInterface + ".zone.getActiveZones"
	if err := c.call(method, &zones); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}

	slog.Debug("active zones listed", "count", len(zones))
	return zones, nil
}

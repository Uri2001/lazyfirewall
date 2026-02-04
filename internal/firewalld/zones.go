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

	var raw interface{}
	method := dbusInterface + ".zone.getActiveZones"
	if err := c.call(method, &raw); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}

	zones, err := normalizeActiveZones(raw)
	if err != nil {
		return nil, err
	}
	slog.Debug("active zones listed", "count", len(zones))
	return zones, nil
}

func normalizeActiveZones(raw interface{}) (map[string][]string, error) {
	switch val := raw.(type) {
	case map[string][]string:
		return val, nil
	case map[string]map[string][]string:
		return flattenActiveZones(val), nil
	case map[string]map[string]dbus.Variant:
		out := make(map[string][]string, len(val))
		for zone, data := range val {
			list := make([]string, 0)
			if v, ok := data["interfaces"]; ok {
				list = append(list, variantToStringSlice(v)...)
			}
			if v, ok := data["sources"]; ok {
				list = append(list, variantToStringSlice(v)...)
			}
			out[zone] = dedupeStrings(list)
		}
		return out, nil
	case map[string]dbus.Variant:
		out := make(map[string][]string, len(val))
		for zone, v := range val {
			out[zone] = extractZoneRefs(v)
		}
		return out, nil
	case dbus.Variant:
		return normalizeActiveZones(val.Value())
	default:
		return nil, fmt.Errorf("unexpected active zones format: %T", raw)
	}
}

func flattenActiveZones(input map[string]map[string][]string) map[string][]string {
	out := make(map[string][]string, len(input))
	for zone, data := range input {
		list := make([]string, 0)
		ifaces, ok := data["interfaces"]
		if ok {
			list = append(list, ifaces...)
		}
		sources, ok := data["sources"]
		if ok {
			list = append(list, sources...)
		}
		out[zone] = dedupeStrings(list)
	}
	return out
}

func extractZoneRefs(v dbus.Variant) []string {
	return dedupeStrings(toStringSlice(v.Value()))
}

func toStringSlice(value interface{}) []string {
	switch val := value.(type) {
	case []string:
		return val
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case dbus.Variant:
		return variantToStringSlice(val)
	case map[string][]string:
		list := make([]string, 0)
		for _, items := range val {
			list = append(list, items...)
		}
		return list
	case map[string]interface{}:
		list := make([]string, 0)
		for _, item := range val {
			list = append(list, toStringSlice(item)...)
		}
		return list
	default:
		return nil
	}
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return items
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

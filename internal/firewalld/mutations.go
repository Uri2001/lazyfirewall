//go:build linux
// +build linux

package firewalld

import "log/slog"

func (c *Client) AddServicePermanent(zone, service string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding service (permanent)", "zone", zone, "service", service)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.addService"
	return c.callObject(obj, method, nil, service)
}

func (c *Client) AddServiceRuntime(zone, service string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding service (runtime)", "zone", zone, "service", service)
	method := dbusInterface + ".zone.addService"
	return c.call(method, nil, zone, service, uint32(0))
}

func (c *Client) RemoveServicePermanent(zone, service string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing service (permanent)", "zone", zone, "service", service)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.removeService"
	return c.callObject(obj, method, nil, service)
}

func (c *Client) RemoveServiceRuntime(zone, service string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing service (runtime)", "zone", zone, "service", service)
	method := dbusInterface + ".zone.removeService"
	return c.call(method, nil, zone, service)
}

func (c *Client) AddPortPermanent(zone string, port Port) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding port (permanent)", "zone", zone, "port", port.Port, "protocol", port.Protocol)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.addPort"
	return c.callObject(obj, method, nil, port.Port, port.Protocol)
}

func (c *Client) AddPortRuntime(zone string, port Port) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding port (runtime)", "zone", zone, "port", port.Port, "protocol", port.Protocol)
	method := dbusInterface + ".zone.addPort"
	return c.call(method, nil, zone, port.Port, port.Protocol, uint32(0))
}

func (c *Client) RemovePortPermanent(zone string, port Port) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing port (permanent)", "zone", zone, "port", port.Port, "protocol", port.Protocol)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.removePort"
	return c.callObject(obj, method, nil, port.Port, port.Protocol)
}

func (c *Client) RemovePortRuntime(zone string, port Port) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing port (runtime)", "zone", zone, "port", port.Port, "protocol", port.Protocol)
	method := dbusInterface + ".zone.removePort"
	return c.call(method, nil, zone, port.Port, port.Protocol)
}

func (c *Client) AddRichRulePermanent(zone, rule string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding rich rule (permanent)", "zone", zone, "rule", rule)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.addRichRule"
	return c.callObject(obj, method, nil, rule)
}

func (c *Client) AddRichRuleRuntime(zone, rule string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding rich rule (runtime)", "zone", zone, "rule", rule)
	method := dbusInterface + ".zone.addRichRule"
	return c.call(method, nil, zone, rule, uint32(0))
}

func (c *Client) RemoveRichRulePermanent(zone, rule string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing rich rule (permanent)", "zone", zone, "rule", rule)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.removeRichRule"
	return c.callObject(obj, method, nil, rule)
}

func (c *Client) RemoveRichRuleRuntime(zone, rule string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing rich rule (runtime)", "zone", zone, "rule", rule)
	method := dbusInterface + ".zone.removeRichRule"
	return c.call(method, nil, zone, rule)
}

func (c *Client) AddInterfacePermanent(zone, iface string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding interface (permanent)", "zone", zone, "interface", iface)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.addInterface"
	return c.callObject(obj, method, nil, iface)
}

func (c *Client) AddInterfaceRuntime(zone, iface string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding interface (runtime)", "zone", zone, "interface", iface)
	method := dbusInterface + ".zone.addInterface"
	return c.call(method, nil, zone, iface, uint32(0))
}

func (c *Client) RemoveInterfacePermanent(zone, iface string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing interface (permanent)", "zone", zone, "interface", iface)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.removeInterface"
	return c.callObject(obj, method, nil, iface)
}

func (c *Client) RemoveInterfaceRuntime(zone, iface string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing interface (runtime)", "zone", zone, "interface", iface)
	method := dbusInterface + ".zone.removeInterface"
	return c.call(method, nil, zone, iface)
}

func (c *Client) AddSourcePermanent(zone, source string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding source (permanent)", "zone", zone, "source", source)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.addSource"
	return c.callObject(obj, method, nil, source)
}

func (c *Client) AddSourceRuntime(zone, source string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("adding source (runtime)", "zone", zone, "source", source)
	method := dbusInterface + ".zone.addSource"
	return c.call(method, nil, zone, source, uint32(0))
}

func (c *Client) RemoveSourcePermanent(zone, source string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing source (permanent)", "zone", zone, "source", source)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.removeSource"
	return c.callObject(obj, method, nil, source)
}

func (c *Client) RemoveSourceRuntime(zone, source string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("removing source (runtime)", "zone", zone, "source", source)
	method := dbusInterface + ".zone.removeSource"
	return c.call(method, nil, zone, source)
}

func (c *Client) EnableMasqueradePermanent(zone string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("enable masquerade (permanent)", "zone", zone)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.addMasquerade"
	return c.callObject(obj, method, nil)
}

func (c *Client) DisableMasqueradePermanent(zone string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("disable masquerade (permanent)", "zone", zone)
	obj, err := c.getConfigZoneObject(zone)
	if err != nil {
		return err
	}

	method := dbusInterface + ".config.zone.removeMasquerade"
	return c.callObject(obj, method, nil)
}

func (c *Client) EnableMasqueradeRuntime(zone string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("enable masquerade (runtime)", "zone", zone)
	method := dbusInterface + ".zone.addMasquerade"
	return c.call(method, nil, zone, uint32(0))
}

func (c *Client) DisableMasqueradeRuntime(zone string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("disable masquerade (runtime)", "zone", zone)
	method := dbusInterface + ".zone.removeMasquerade"
	return c.call(method, nil, zone)
}

func (c *Client) RuntimeToPermanent() error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("committing runtime to permanent")
	method := dbusInterface + ".runtimeToPermanent"
	if err := c.call(method, nil); err != nil {
		return err
	}
	c.invalidateAllIPSetEntriesCache()
	return nil
}

func (c *Client) Reload() error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Info("reloading firewalld")
	method := dbusInterface + ".reload"
	if err := c.call(method, nil); err != nil {
		return err
	}
	c.invalidateAllIPSetEntriesCache()
	return nil
}

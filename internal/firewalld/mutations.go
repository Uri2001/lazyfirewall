//go:build linux
// +build linux

package firewalld

import "log/slog"

func (c *Client) AddServicePermanent(zone, service string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
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

	slog.Info("adding service (runtime)", "zone", zone, "service", service)
	method := dbusInterface + ".zone.addService"
	return c.call(method, nil, zone, service, uint32(0))
}

func (c *Client) RemoveServicePermanent(zone, service string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
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

	slog.Info("removing service (runtime)", "zone", zone, "service", service)
	method := dbusInterface + ".zone.removeService"
	return c.call(method, nil, zone, service)
}

func (c *Client) AddPortPermanent(zone string, port Port) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
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

	slog.Info("adding port (runtime)", "zone", zone, "port", port.Port, "protocol", port.Protocol)
	method := dbusInterface + ".zone.addPort"
	return c.call(method, nil, zone, port.Port, port.Protocol, uint32(0))
}

func (c *Client) RemovePortPermanent(zone string, port Port) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
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

	slog.Info("removing port (runtime)", "zone", zone, "port", port.Port, "protocol", port.Protocol)
	method := dbusInterface + ".zone.removePort"
	return c.call(method, nil, zone, port.Port, port.Protocol)
}

func (c *Client) RuntimeToPermanent() error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}

	slog.Info("committing runtime to permanent")
	method := dbusInterface + ".runtimeToPermanent"
	return c.call(method, nil)
}

func (c *Client) Reload() error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}

	slog.Info("reloading firewalld")
	method := dbusInterface + ".reload"
	return c.call(method, nil)
}

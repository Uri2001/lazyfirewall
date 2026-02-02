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

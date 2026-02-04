//go:build linux
// +build linux

package firewalld

import "log/slog"

func (c *Client) QueryPanicMode() (bool, error) {
	if c.apiVersion != APIv2 {
		return false, ErrUnsupportedAPI
	}

	var enabled bool
	method := dbusInterface + ".queryPanicMode"
	if err := c.call(method, &enabled); err != nil {
		if isPermissionDenied(err) {
			return false, ErrPermissionDenied
		}
		return false, err
	}

	return enabled, nil
}

func (c *Client) EnablePanicMode() error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Warn("enabling panic mode")
	method := dbusInterface + ".enablePanicMode"
	return c.call(method, nil)
}

func (c *Client) DisablePanicMode() error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}

	slog.Warn("disabling panic mode")
	method := dbusInterface + ".disablePanicMode"
	return c.call(method, nil)
}

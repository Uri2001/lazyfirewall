//go:build linux
// +build linux

package firewalld

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

type APIVersion int

const (
	APIUnknown APIVersion = iota
	APIv1
	APIv2
)

func (c *Client) detectVersion() error {
	var v dbus.Variant
	if err := c.call("org.freedesktop.DBus.Properties.Get", &v, dbusInterface, "version"); err != nil {
		slog.Warn("failed to detect version, assuming v2", "error", err)
		c.version = "unknown"
		c.apiVersion = APIv2
		return nil
	}

	version, ok := v.Value().(string)
	if !ok || version == "" {
		slog.Warn("failed to parse version, assuming v2", "value", fmt.Sprintf("%v", v.Value()))
		c.version = "unknown"
		c.apiVersion = APIv2
		return nil
	}

	c.version = version
	c.apiVersion = parseVersion(version)

	slog.Info("firewalld detected", "version", version, "api", c.apiVersion)
	return nil
}

func parseVersion(version string) APIVersion {
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return APIv2
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 1 {
		return APIv1
	}

	return APIv2
}

func (v APIVersion) String() string {
	switch v {
	case APIv1:
		return "v1 (firewalld 0.x)"
	case APIv2:
		return "v2 (firewalld 1.x+)"
	default:
		return "unknown"
	}
}

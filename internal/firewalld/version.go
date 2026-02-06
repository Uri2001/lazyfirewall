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
		return fmt.Errorf("failed to detect firewalld version: %w", err)
	}

	version, ok := v.Value().(string)
	if !ok {
		return fmt.Errorf("invalid version type: %T (expected string)", v.Value())
	}
	if version == "" {
		return fmt.Errorf("empty version string returned")
	}

	c.version = version
	c.apiVersion = parseVersion(version)
	if c.apiVersion == APIUnknown {
		slog.Warn("unknown firewalld version, falling back to v2 API", "version", version)
		c.apiVersion = APIv2
	}

	slog.Info("firewalld detected", "version", version, "api", c.apiVersion)
	return nil
}

func parseVersion(version string) APIVersion {
	parts := strings.Split(version, ".")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return APIUnknown
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return APIUnknown
	}
	if major < 1 {
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

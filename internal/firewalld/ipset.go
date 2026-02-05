//go:build linux
// +build linux

package firewalld

import (
	"errors"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

type ipsetSettings struct {
	Version     string
	Short       string
	Description string
	Type        string
	Options     map[string]string
	Entries     []string
}

func (c *Client) ListIPSets(permanent bool) ([]string, error) {
	if c.apiVersion != APIv2 {
		return nil, ErrUnsupportedAPI
	}
	if permanent {
		return c.listIPSetsPermanent()
	}
	return c.listIPSetsRuntime()
}

func (c *Client) listIPSetsRuntime() ([]string, error) {
	slog.Debug("listing ipsets (runtime)")
	var sets []string
	method := dbusInterface + ".ipset.getIPSets"
	if err := c.call(method, &sets); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}
	sort.Strings(sets)
	return sets, nil
}

func (c *Client) listIPSetsPermanent() ([]string, error) {
	slog.Debug("listing ipsets (permanent)")
	var paths []dbus.ObjectPath
	method := dbusInterface + ".config.listIPSets"
	configObj := c.conn.Object(dbusInterface, dbusConfigPath)
	if err := c.callObject(configObj, method, &paths); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}
	sets := make([]string, 0, len(paths))
	for _, p := range paths {
		name := path.Base(string(p))
		if name != "" {
			sets = append(sets, name)
		}
	}
	sort.Strings(sets)
	return sets, nil
}

func (c *Client) GetIPSetEntries(name string, permanent bool) ([]string, error) {
	if c.apiVersion != APIv2 {
		return nil, ErrUnsupportedAPI
	}
	if permanent {
		return c.getIPSetEntriesPermanent(name)
	}
	return c.getIPSetEntriesRuntime(name)
}

func (c *Client) getIPSetEntriesRuntime(name string) ([]string, error) {
	slog.Debug("fetching ipset entries (runtime)", "ipset", name)
	var entries []string
	method := dbusInterface + ".ipset.getEntries"
	if err := c.call(method, &entries, name); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		if isInvalidIPSet(err) {
			return nil, ErrInvalidIPSet
		}
		return nil, err
	}
	return entries, nil
}

func (c *Client) getIPSetEntriesPermanent(name string) ([]string, error) {
	slog.Debug("fetching ipset entries (permanent)", "ipset", name)
	var entries []string
	obj := c.configIPSetObject(name)
	method := dbusInterface + ".config.ipset.getEntries"
	if err := c.callObject(obj, method, &entries); err != nil {
		if isPermissionDenied(err) {
			return nil, ErrPermissionDenied
		}
		if isInvalidIPSet(err) {
			return nil, ErrInvalidIPSet
		}
		return nil, err
	}
	return entries, nil
}

func (c *Client) AddIPSetPermanent(name, ipsetType string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}
	if name == "" {
		return fmt.Errorf("ipset name is empty")
	}
	if ipsetType == "" {
		return fmt.Errorf("ipset type is empty")
	}

	slog.Info("adding ipset (permanent)", "ipset", name, "type", ipsetType)
	settings := ipsetSettings{
		Version:     "",
		Short:       name,
		Description: "",
		Type:        ipsetType,
		Options:     map[string]string{},
		Entries:     nil,
	}
	method := dbusInterface + ".config.addIPSet"
	configObj := c.conn.Object(dbusInterface, dbusConfigPath)
	var path dbus.ObjectPath
	if err := c.callObject(configObj, method, &path, name, settings); err != nil {
		if isPermissionDenied(err) {
			return ErrPermissionDenied
		}
		return err
	}
	return nil
}

func (c *Client) RemoveIPSetPermanent(name string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}
	if name == "" {
		return fmt.Errorf("ipset name is empty")
	}

	slog.Info("removing ipset (permanent)", "ipset", name)
	obj := c.configIPSetObject(name)
	method := dbusInterface + ".config.ipset.remove"
	if err := c.callObject(obj, method, nil); err != nil {
		if isPermissionDenied(err) {
			return ErrPermissionDenied
		}
		if isInvalidIPSet(err) {
			return ErrInvalidIPSet
		}
		return err
	}
	return nil
}

func (c *Client) AddIPSetEntryRuntime(name, entry string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}
	slog.Info("adding ipset entry (runtime)", "ipset", name, "entry", entry)
	method := dbusInterface + ".ipset.addEntry"
	return c.call(method, nil, name, entry)
}

func (c *Client) RemoveIPSetEntryRuntime(name, entry string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}
	slog.Info("removing ipset entry (runtime)", "ipset", name, "entry", entry)
	method := dbusInterface + ".ipset.removeEntry"
	return c.call(method, nil, name, entry)
}

func (c *Client) AddIPSetEntryPermanent(name, entry string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}
	slog.Info("adding ipset entry (permanent)", "ipset", name, "entry", entry)
	obj := c.configIPSetObject(name)
	method := dbusInterface + ".config.ipset.addEntry"
	return c.callObject(obj, method, nil, entry)
}

func (c *Client) RemoveIPSetEntryPermanent(name, entry string) error {
	if c.apiVersion != APIv2 {
		return ErrUnsupportedAPI
	}
	if c.readOnly {
		return ErrPermissionDenied
	}
	slog.Info("removing ipset entry (permanent)", "ipset", name, "entry", entry)
	obj := c.configIPSetObject(name)
	method := dbusInterface + ".config.ipset.removeEntry"
	return c.callObject(obj, method, nil, entry)
}

func (c *Client) configIPSetObject(name string) dbus.BusObject {
	path := dbus.ObjectPath(dbusConfigPath + "/ipset/" + name)
	return c.conn.Object(dbusInterface, path)
}

func isInvalidIPSet(err error) bool {
	var dbusErr *dbus.Error
	if errors.As(err, &dbusErr) {
		name := strings.ToLower(dbusErr.Name)
		if strings.Contains(name, "invalid_ipset") || strings.Contains(name, "invalidipset") {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid_ipset") || strings.Contains(msg, "invalid ipset") || strings.Contains(msg, "invalidipset")
}

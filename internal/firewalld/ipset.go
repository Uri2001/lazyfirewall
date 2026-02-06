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
	"time"

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

const ipsetEntriesCacheTTL = 5 * time.Second

type ipsetEntriesCacheKey struct {
	name      string
	permanent bool
}

type ipsetEntriesCacheEntry struct {
	entries   []string
	expiresAt time.Time
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
	if cached, ok := c.getCachedIPSetEntries(name, permanent); ok {
		return cached, nil
	}

	var (
		entries []string
		err     error
	)
	if permanent {
		entries, err = c.getIPSetEntriesPermanent(name)
	} else {
		entries, err = c.getIPSetEntriesRuntime(name)
	}
	if err != nil {
		return nil, err
	}
	c.putIPSetEntriesCache(name, permanent, entries)
	return entries, nil
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
	c.invalidateIPSetEntriesCache(name, true)
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
	c.invalidateIPSetEntriesCache(name, true)
	c.invalidateIPSetEntriesCache(name, false)
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
	if err := c.call(method, nil, name, entry); err != nil {
		return err
	}
	c.invalidateIPSetEntriesCache(name, false)
	return nil
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
	if err := c.call(method, nil, name, entry); err != nil {
		return err
	}
	c.invalidateIPSetEntriesCache(name, false)
	return nil
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
	if err := c.callObject(obj, method, nil, entry); err != nil {
		return err
	}
	c.invalidateIPSetEntriesCache(name, true)
	return nil
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
	if err := c.callObject(obj, method, nil, entry); err != nil {
		return err
	}
	c.invalidateIPSetEntriesCache(name, true)
	return nil
}

func (c *Client) getCachedIPSetEntries(name string, permanent bool) ([]string, bool) {
	key := ipsetEntriesCacheKey{name: name, permanent: permanent}
	now := time.Now()

	c.ipsetEntriesMu.RLock()
	entry, ok := c.ipsetEntriesCache[key]
	c.ipsetEntriesMu.RUnlock()
	if !ok {
		return nil, false
	}
	if now.After(entry.expiresAt) {
		c.ipsetEntriesMu.Lock()
		delete(c.ipsetEntriesCache, key)
		c.ipsetEntriesMu.Unlock()
		return nil, false
	}

	copied := make([]string, len(entry.entries))
	copy(copied, entry.entries)
	return copied, true
}

func (c *Client) putIPSetEntriesCache(name string, permanent bool, entries []string) {
	copied := make([]string, len(entries))
	copy(copied, entries)

	key := ipsetEntriesCacheKey{name: name, permanent: permanent}
	c.ipsetEntriesMu.Lock()
	if c.ipsetEntriesCache == nil {
		c.ipsetEntriesCache = make(map[ipsetEntriesCacheKey]ipsetEntriesCacheEntry)
	}
	c.ipsetEntriesCache[key] = ipsetEntriesCacheEntry{
		entries:   copied,
		expiresAt: time.Now().Add(ipsetEntriesCacheTTL),
	}
	c.ipsetEntriesMu.Unlock()
}

func (c *Client) invalidateIPSetEntriesCache(name string, permanent bool) {
	key := ipsetEntriesCacheKey{name: name, permanent: permanent}
	c.ipsetEntriesMu.Lock()
	delete(c.ipsetEntriesCache, key)
	c.ipsetEntriesMu.Unlock()
}

func (c *Client) invalidateAllIPSetEntriesCache() {
	c.ipsetEntriesMu.Lock()
	c.ipsetEntriesCache = make(map[ipsetEntriesCacheKey]ipsetEntriesCacheEntry)
	c.ipsetEntriesMu.Unlock()
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

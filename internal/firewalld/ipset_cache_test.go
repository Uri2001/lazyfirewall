//go:build linux
// +build linux

package firewalld

import (
	"testing"
	"time"
)

func TestIPSetEntriesCacheCopyIsolation(t *testing.T) {
	c := &Client{}
	src := []string{"1.1.1.1", "2.2.2.2"}
	c.putIPSetEntriesCache("blocklist", true, src)

	src[0] = "mutated-src"
	got, ok := c.getCachedIPSetEntries("blocklist", true)
	if !ok {
		t.Fatalf("getCachedIPSetEntries() cache miss, want hit")
	}
	if got[0] != "1.1.1.1" {
		t.Fatalf("cached value mutated by source slice: %#v", got)
	}

	got[0] = "mutated-read"
	got2, ok := c.getCachedIPSetEntries("blocklist", true)
	if !ok {
		t.Fatalf("getCachedIPSetEntries() second read miss, want hit")
	}
	if got2[0] != "1.1.1.1" {
		t.Fatalf("cached value mutated by returned slice: %#v", got2)
	}
}

func TestIPSetEntriesCacheExpiry(t *testing.T) {
	c := &Client{}
	c.putIPSetEntriesCache("set1", false, []string{"10.0.0.1"})

	key := ipsetEntriesCacheKey{name: "set1", permanent: false}
	c.ipsetEntriesMu.Lock()
	entry := c.ipsetEntriesCache[key]
	entry.expiresAt = time.Now().Add(-time.Second)
	c.ipsetEntriesCache[key] = entry
	c.ipsetEntriesMu.Unlock()

	if _, ok := c.getCachedIPSetEntries("set1", false); ok {
		t.Fatalf("expired cache entry should not be returned")
	}

	c.ipsetEntriesMu.RLock()
	_, stillPresent := c.ipsetEntriesCache[key]
	c.ipsetEntriesMu.RUnlock()
	if stillPresent {
		t.Fatalf("expired cache entry should be removed")
	}
}

func TestIPSetEntriesCacheInvalidation(t *testing.T) {
	c := &Client{}
	c.putIPSetEntriesCache("set1", false, []string{"r1"})
	c.putIPSetEntriesCache("set1", true, []string{"p1"})

	c.invalidateIPSetEntriesCache("set1", false)
	if _, ok := c.getCachedIPSetEntries("set1", false); ok {
		t.Fatalf("runtime cache should be invalidated")
	}
	if _, ok := c.getCachedIPSetEntries("set1", true); !ok {
		t.Fatalf("permanent cache should remain")
	}

	c.invalidateAllIPSetEntriesCache()
	if _, ok := c.getCachedIPSetEntries("set1", true); ok {
		t.Fatalf("all cache should be invalidated")
	}
}

func TestGetIPSetEntriesReturnsCachedValue(t *testing.T) {
	c := &Client{apiVersion: APIv2}
	c.putIPSetEntriesCache("set1", true, []string{"192.168.1.1"})

	got, err := c.GetIPSetEntries("set1", true)
	if err != nil {
		t.Fatalf("GetIPSetEntries() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != "192.168.1.1" {
		t.Fatalf("GetIPSetEntries() = %#v, want [192.168.1.1]", got)
	}
}

//go:build linux
// +build linux

package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	timeFormat   = "20060102-150405"
	keepBackups  = 10
	backupFolder = ".config/lazyfirewall/backups"
)

type Backup struct {
	Path string
	Zone string
	Time time.Time
	Size int64
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, backupFolder), nil
}

func CreateZoneBackup(zone string) (Backup, error) {
	src, err := zoneFilePath(zone)
	if err != nil {
		return Backup{}, err
	}
	dir, err := Dir()
	if err != nil {
		return Backup{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Backup{}, err
	}

	ts := time.Now()
	name := fmt.Sprintf("zone-%s-%s.xml", zone, ts.Format(timeFormat))
	dest := filepath.Join(dir, name)
	if err := copyFile(src, dest); err != nil {
		return Backup{}, err
	}

	info, err := os.Stat(dest)
	if err != nil {
		return Backup{}, err
	}

	b := Backup{
		Path: dest,
		Zone: zone,
		Time: ts,
		Size: info.Size(),
	}
	_ = pruneBackups(zone, keepBackups)
	return b, nil
}

func ListBackups(zone string) ([]Backup, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	prefix := "zone-" + zone + "-"
	items := make([]Backup, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".xml") {
			continue
		}
		tsPart := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".xml")
		ts, err := time.Parse(timeFormat, tsPart)
		if err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, Backup{
			Path: filepath.Join(dir, name),
			Zone: zone,
			Time: ts,
			Size: info.Size(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Time.After(items[j].Time)
	})
	return items, nil
}

func RestoreZoneBackup(zone string, b Backup) error {
	if b.Path == "" {
		return fmt.Errorf("backup path is empty")
	}
	dir := filepath.Join("/etc/firewalld/zones")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(dir, zone+".xml")
	return copyFile(b.Path, dest)
}

func zoneFilePath(zone string) (string, error) {
	etc := filepath.Join("/etc/firewalld/zones", zone+".xml")
	if fileExists(etc) {
		return etc, nil
	}
	usr := filepath.Join("/usr/lib/firewalld/zones", zone+".xml")
	if fileExists(usr) {
		return usr, nil
	}
	return "", os.ErrNotExist
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func pruneBackups(zone string, keep int) error {
	if keep <= 0 {
		return nil
	}
	items, err := ListBackups(zone)
	if err != nil {
		return err
	}
	if len(items) <= keep {
		return nil
	}
	for _, b := range items[keep:] {
		_ = os.Remove(b.Path)
	}
	return nil
}

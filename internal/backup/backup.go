//go:build linux
// +build linux

package backup

import (
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"lazyfirewall/internal/validation"
)

const (
	timeFormat   = "20060102-150405"
	keepBackups  = 10
	backupFolder = ".config/lazyfirewall/backups"
)

var (
	zoneConfigDir = "/etc/firewalld/zones"
	zoneSystemDir = "/usr/lib/firewalld/zones"
)

type Backup struct {
	Path        string
	Zone        string
	Time        time.Time
	Size        int64
	Description string
}

func Dir() (string, error) {
	home, err := resolveHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, backupFolder), nil
}

func resolveHomeDir() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" && sudoUser != "root" {
		u, err := user.Lookup(sudoUser)
		if err == nil && u.HomeDir != "" {
			return u.HomeDir, nil
		}
	}
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}
	return os.UserHomeDir()
}

func CreateZoneBackup(zone string) (Backup, error) {
	return CreateZoneBackupWithDescription(zone, "")
}

func CreateZoneBackupWithDescription(zone, description string) (Backup, error) {
	if err := validation.IsValidZoneName(zone); err != nil {
		return Backup{}, fmt.Errorf("invalid zone name: %w", err)
	}

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
	suffix := ""
	desc := strings.TrimSpace(description)
	if desc != "" {
		desc = truncateDescription(desc, 40)
		suffix = "__" + url.PathEscape(desc)
	}
	name := fmt.Sprintf("zone-%s-%s%s.xml", zone, ts.Format(timeFormat), suffix)
	dest := filepath.Join(dir, name)
	if err := copyFile(src, dest); err != nil {
		return Backup{}, err
	}
	slog.Info("backup created", "zone", zone, "src", src, "dest", dest)

	info, err := os.Stat(dest)
	if err != nil {
		return Backup{}, err
	}

	b := Backup{
		Path:        dest,
		Zone:        zone,
		Time:        ts,
		Size:        info.Size(),
		Description: desc,
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
		desc := ""
		if parts := strings.SplitN(tsPart, "__", 2); len(parts) == 2 {
			tsPart = parts[0]
			if decoded, err := url.PathUnescape(parts[1]); err == nil {
				desc = decoded
			} else {
				desc = parts[1]
			}
		}
		ts, err := time.Parse(timeFormat, tsPart)
		if err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, Backup{
			Path:        filepath.Join(dir, name),
			Zone:        zone,
			Time:        ts,
			Size:        info.Size(),
			Description: desc,
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
	if err := validation.IsValidZoneName(zone); err != nil {
		return fmt.Errorf("invalid zone name: %w", err)
	}

	dir := zoneConfigDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(dir, zone+".xml")

	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	tempDest := dest + ".tmp." + timestamp
	preRestoreBackup := dest + ".pre-restore." + timestamp

	defer func() {
		_ = os.Remove(tempDest)
	}()

	if fileExists(dest) {
		if err := copyFile(dest, preRestoreBackup); err != nil {
			return fmt.Errorf("failed to create pre-restore backup: %w", err)
		}
	}

	if err := copyFile(b.Path, tempDest); err != nil {
		if fileExists(preRestoreBackup) {
			_ = os.Remove(preRestoreBackup)
		}
		return fmt.Errorf("failed to copy backup to temp file: %w", err)
	}

	if err := os.Rename(tempDest, dest); err != nil {
		_ = os.Remove(tempDest)
		if fileExists(preRestoreBackup) {
			_ = os.Remove(preRestoreBackup)
		}
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	slog.Info("zone file restored", "zone", zone, "from", b.Path, "to", dest)
	return nil
}

func zoneFilePath(zone string) (string, error) {
	if err := validation.IsValidZoneName(zone); err != nil {
		return "", fmt.Errorf("invalid zone name: %w", err)
	}

	etc := filepath.Join(zoneConfigDir, zone+".xml")
	if fileExists(etc) {
		return etc, nil
	}
	usr := filepath.Join(zoneSystemDir, zone+".xml")
	if fileExists(usr) {
		return usr, nil
	}
	return "", os.ErrNotExist
}

func ZoneDestinationPath(zone string) (string, error) {
	if err := validation.IsValidZoneName(zone); err != nil {
		return "", fmt.Errorf("invalid zone name: %w", err)
	}
	return filepath.Join(zoneConfigDir, zone+".xml"), nil
}

func GetPreRestoreBackupPath(zone string) (string, error) {
	if err := validation.IsValidZoneName(zone); err != nil {
		return "", fmt.Errorf("invalid zone name: %w", err)
	}

	pattern := filepath.Join(zoneConfigDir, zone+".xml.pre-restore.*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", nil
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func CleanupPreRestoreBackup(zone string) error {
	path, err := GetPreRestoreBackupPath(zone)
	if err != nil || path == "" {
		return err
	}
	return os.Remove(path)
}

func CopyFile(src, dest string) error {
	return copyFile(src, dest)
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

func truncateDescription(desc string, max int) string {
	if max <= 0 || desc == "" {
		return ""
	}
	if utf8.RuneCountInString(desc) <= max {
		return desc
	}
	runes := []rune(desc)
	return string(runes[:max])
}

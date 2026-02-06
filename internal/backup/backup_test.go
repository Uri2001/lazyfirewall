//go:build linux
// +build linux

package backup

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func withZoneDirs(t *testing.T, configDir, systemDir string) {
	t.Helper()
	oldConfigDir := zoneConfigDir
	oldSystemDir := zoneSystemDir
	zoneConfigDir = configDir
	zoneSystemDir = systemDir
	t.Cleanup(func() {
		zoneConfigDir = oldConfigDir
		zoneSystemDir = oldSystemDir
	})
}

func TestCreateZoneBackupWithDescription_InvalidZone(t *testing.T) {
	_, err := CreateZoneBackupWithDescription("../bad", "desc")
	if err == nil {
		t.Fatalf("expected validation error for invalid zone")
	}
}

func TestCreateZoneBackupWithDescription(t *testing.T) {
	tempDir := t.TempDir()
	withZoneDirs(t, filepath.Join(tempDir, "etc-zones"), filepath.Join(tempDir, "usr-zones"))

	if err := os.MkdirAll(zoneConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir zone dir: %v", err)
	}
	zonePath := filepath.Join(zoneConfigDir, "public.xml")
	zoneData := []byte(`<?xml version="1.0"?><zone><short>Public</short></zone>`)
	if err := os.WriteFile(zonePath, zoneData, 0o644); err != nil {
		t.Fatalf("write zone file: %v", err)
	}

	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})

	b, err := CreateZoneBackupWithDescription("public", "  release backup  ")
	if err != nil {
		t.Fatalf("CreateZoneBackupWithDescription() error = %v", err)
	}
	if b.Zone != "public" {
		t.Fatalf("backup zone = %q, want %q", b.Zone, "public")
	}
	if b.Description != "release backup" {
		t.Fatalf("backup description = %q, want %q", b.Description, "release backup")
	}
	if !fileExists(b.Path) {
		t.Fatalf("backup file was not created: %s", b.Path)
	}
}

func TestRestoreZoneBackup_Transactional(t *testing.T) {
	tempDir := t.TempDir()
	withZoneDirs(t, filepath.Join(tempDir, "etc-zones"), filepath.Join(tempDir, "usr-zones"))

	if err := os.MkdirAll(zoneConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir zone dir: %v", err)
	}

	dest := filepath.Join(zoneConfigDir, "public.xml")
	oldData := []byte("old")
	if err := os.WriteFile(dest, oldData, 0o644); err != nil {
		t.Fatalf("write old zone file: %v", err)
	}

	backupPath := filepath.Join(tempDir, "backup.xml")
	newData := []byte("new")
	if err := os.WriteFile(backupPath, newData, 0o644); err != nil {
		t.Fatalf("write backup file: %v", err)
	}

	if err := RestoreZoneBackup("public", Backup{Path: backupPath}); err != nil {
		t.Fatalf("RestoreZoneBackup() error = %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(got) != string(newData) {
		t.Fatalf("restored content = %q, want %q", string(got), string(newData))
	}

	preRestorePath, err := GetPreRestoreBackupPath("public")
	if err != nil {
		t.Fatalf("GetPreRestoreBackupPath() error = %v", err)
	}
	if preRestorePath == "" {
		t.Fatalf("expected pre-restore backup path")
	}

	preData, err := os.ReadFile(preRestorePath)
	if err != nil {
		t.Fatalf("read pre-restore backup: %v", err)
	}
	if string(preData) != string(oldData) {
		t.Fatalf("pre-restore content = %q, want %q", string(preData), string(oldData))
	}
}

func TestListBackups_SortedAndDecodedDescription(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}

	t1 := time.Now().Add(-1 * time.Hour).Format(timeFormat)
	t2 := time.Now().Format(timeFormat)
	f1 := filepath.Join(dir, "zone-public-"+t1+"__first%20backup.xml")
	f2 := filepath.Join(dir, "zone-public-"+t2+"__second%20backup.xml")
	if err := os.WriteFile(f1, []byte("one"), 0o644); err != nil {
		t.Fatalf("write f1: %v", err)
	}
	if err := os.WriteFile(f2, []byte("two"), 0o644); err != nil {
		t.Fatalf("write f2: %v", err)
	}

	items, err := ListBackups("public")
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(items))
	}
	if !items[0].Time.After(items[1].Time) {
		t.Fatalf("backups are not sorted descending by time")
	}
	if items[0].Description != "second backup" {
		t.Fatalf("description = %q, want %q", items[0].Description, "second backup")
	}
}

func TestGetAndCleanupPreRestoreBackup(t *testing.T) {
	tempDir := t.TempDir()
	withZoneDirs(t, filepath.Join(tempDir, "etc-zones"), filepath.Join(tempDir, "usr-zones"))
	if err := os.MkdirAll(zoneConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir zone dir: %v", err)
	}

	path1 := filepath.Join(zoneConfigDir, "public.xml.pre-restore."+strconv.FormatInt(time.Now().UnixNano(), 10))
	time.Sleep(1 * time.Millisecond)
	path2 := filepath.Join(zoneConfigDir, "public.xml.pre-restore."+strconv.FormatInt(time.Now().UnixNano(), 10))
	if err := os.WriteFile(path1, []byte("old1"), 0o644); err != nil {
		t.Fatalf("write path1: %v", err)
	}
	if err := os.WriteFile(path2, []byte("old2"), 0o644); err != nil {
		t.Fatalf("write path2: %v", err)
	}

	got, err := GetPreRestoreBackupPath("public")
	if err != nil {
		t.Fatalf("GetPreRestoreBackupPath() error = %v", err)
	}
	if got != path2 {
		t.Fatalf("latest pre-restore path = %q, want %q", got, path2)
	}

	if err := CleanupPreRestoreBackup("public"); err != nil {
		t.Fatalf("CleanupPreRestoreBackup() error = %v", err)
	}
	if _, err := os.Stat(path2); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be removed", path2)
	}
}

func TestTruncateDescription(t *testing.T) {
	if got := truncateDescription("short", 10); got != "short" {
		t.Fatalf("truncateDescription short = %q", got)
	}
	if got := truncateDescription("abcdef", 3); got != "abc" {
		t.Fatalf("truncateDescription long = %q, want abc", got)
	}
	if got := truncateDescription("abcdef", 0); got != "" {
		t.Fatalf("truncateDescription max 0 = %q, want empty", got)
	}
}

func TestZoneDestinationPath_InvalidZone(t *testing.T) {
	_, err := ZoneDestinationPath("../bad")
	if err == nil {
		t.Fatalf("expected invalid zone error")
	}
}

//go:build linux
// +build linux

package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lazyfirewall/internal/backup"
	"lazyfirewall/internal/firewalld"
)

type fakeRichRuleUpdater struct {
	removePermanentCalls int
	addPermanentCalls    int
	removeRuntimeCalls   int
	addRuntimeCalls      int

	removePermanentErr error
	addPermanentErr    error
	removeRuntimeErr   error
	addRuntimeErr      error
}

func (f *fakeRichRuleUpdater) RemoveRichRulePermanent(zone, rule string) error {
	f.removePermanentCalls++
	return f.removePermanentErr
}

func (f *fakeRichRuleUpdater) AddRichRulePermanent(zone, rule string) error {
	f.addPermanentCalls++
	return f.addPermanentErr
}

func (f *fakeRichRuleUpdater) RemoveRichRuleRuntime(zone, rule string) error {
	f.removeRuntimeCalls++
	return f.removeRuntimeErr
}

func (f *fakeRichRuleUpdater) AddRichRuleRuntime(zone, rule string) error {
	f.addRuntimeCalls++
	return f.addRuntimeErr
}

func TestUpdateRichRuleTransaction(t *testing.T) {
	tests := []struct {
		name         string
		removeErr    error
		addErr       error
		restoreErr   error
		wantErr      bool
		wantRollback bool
	}{
		{
			name:         "success",
			wantErr:      false,
			wantRollback: false,
		},
		{
			name:         "add fails rollback succeeds",
			addErr:       errors.New("invalid rule"),
			wantErr:      true,
			wantRollback: true,
		},
		{
			name:         "add fails rollback fails",
			addErr:       errors.New("invalid rule"),
			restoreErr:   errors.New("restore failed"),
			wantErr:      true,
			wantRollback: true,
		},
		{
			name:         "remove fails",
			removeErr:    errors.New("remove failed"),
			wantErr:      true,
			wantRollback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rollbackCalled bool
			err := updateRichRuleTransaction(
				func() error { return tt.removeErr },
				func() error { return tt.addErr },
				func() error {
					rollbackCalled = true
					return tt.restoreErr
				},
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("updateRichRuleTransaction() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if rollbackCalled != tt.wantRollback {
				t.Fatalf("rollbackCalled = %v, want %v", rollbackCalled, tt.wantRollback)
			}
		})
	}
}

func TestUpdateRichRuleCmdUsesPermanentMethods(t *testing.T) {
	client := &fakeRichRuleUpdater{}
	cmd := updateRichRuleCmd(client, "public", "old", "new", true, nil, recordNone, false)
	msg := cmd()

	got, ok := msg.(mutationMsg)
	if !ok {
		t.Fatalf("msg type = %T, want mutationMsg", msg)
	}
	if got.err != nil {
		t.Fatalf("mutationMsg.err = %v, want nil", got.err)
	}
	if client.removePermanentCalls != 1 || client.addPermanentCalls != 1 {
		t.Fatalf("permanent calls = remove:%d add:%d, want 1/1", client.removePermanentCalls, client.addPermanentCalls)
	}
	if client.removeRuntimeCalls != 0 || client.addRuntimeCalls != 0 {
		t.Fatalf("runtime calls = remove:%d add:%d, want 0/0", client.removeRuntimeCalls, client.addRuntimeCalls)
	}
}

func TestUpdateRichRuleCmdUsesRuntimeMethods(t *testing.T) {
	client := &fakeRichRuleUpdater{}
	cmd := updateRichRuleCmd(client, "public", "old", "new", false, nil, recordNone, false)
	msg := cmd()

	got, ok := msg.(mutationMsg)
	if !ok {
		t.Fatalf("msg type = %T, want mutationMsg", msg)
	}
	if got.err != nil {
		t.Fatalf("mutationMsg.err = %v, want nil", got.err)
	}
	if client.removeRuntimeCalls != 1 || client.addRuntimeCalls != 1 {
		t.Fatalf("runtime calls = remove:%d add:%d, want 1/1", client.removeRuntimeCalls, client.addRuntimeCalls)
	}
	if client.removePermanentCalls != 0 || client.addPermanentCalls != 0 {
		t.Fatalf("permanent calls = remove:%d add:%d, want 0/0", client.removePermanentCalls, client.addPermanentCalls)
	}
}

func TestUpdateRichRuleCmdRollbackError(t *testing.T) {
	client := &fakeRichRuleUpdater{
		addRuntimeErr: errors.New("invalid syntax"),
	}
	cmd := updateRichRuleCmd(client, "public", "old", "new", false, nil, recordNone, false)
	msg := cmd()

	got, ok := msg.(mutationMsg)
	if !ok {
		t.Fatalf("msg type = %T, want mutationMsg", msg)
	}
	if got.err == nil {
		t.Fatalf("mutationMsg.err = nil, want error")
	}
	if !strings.Contains(got.err.Error(), "old rule restored") {
		t.Fatalf("mutationMsg.err = %v, want rollback message", got.err)
	}
}

func TestAddZoneCmdRejectsInvalidZone(t *testing.T) {
	cmd := addZoneCmd(&firewalld.Client{}, "../etc")
	msg := cmd()
	got, ok := msg.(zonesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want zonesMsg", msg)
	}
	if got.err == nil {
		t.Fatalf("zonesMsg.err = nil, want error")
	}
	if !strings.Contains(got.err.Error(), "invalid zone name") {
		t.Fatalf("zonesMsg.err = %v, want invalid zone name error", got.err)
	}
}

func TestRemoveZoneCmdRejectsInvalidZone(t *testing.T) {
	cmd := removeZoneCmd(&firewalld.Client{}, `..\windows`)
	msg := cmd()
	got, ok := msg.(zonesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want zonesMsg", msg)
	}
	if got.err == nil {
		t.Fatalf("zonesMsg.err = nil, want error")
	}
}

func TestRestoreBackupCmdGuards(t *testing.T) {
	t.Run("invalid zone", func(t *testing.T) {
		cmd := restoreBackupCmd(&firewalld.Client{}, "../bad", backup.Backup{Path: "/tmp/backup.xml"})
		msg := cmd()
		got, ok := msg.(backupRestoreMsg)
		if !ok {
			t.Fatalf("msg type = %T, want backupRestoreMsg", msg)
		}
		if got.err == nil {
			t.Fatalf("backupRestoreMsg.err = nil, want error")
		}
	})

	t.Run("empty backup path", func(t *testing.T) {
		cmd := restoreBackupCmd(&firewalld.Client{}, "public", backup.Backup{})
		msg := cmd()
		got, ok := msg.(backupRestoreMsg)
		if !ok {
			t.Fatalf("msg type = %T, want backupRestoreMsg", msg)
		}
		if got.err == nil {
			t.Fatalf("backupRestoreMsg.err = nil, want error")
		}
	})
}

func TestImportZoneCmdGuards(t *testing.T) {
	t.Run("invalid zone", func(t *testing.T) {
		cmd := importZoneCmd(&firewalld.Client{}, "../bad", "unused.json")
		msg := cmd()
		got, ok := msg.(importMsg)
		if !ok {
			t.Fatalf("msg type = %T, want importMsg", msg)
		}
		if got.err == nil {
			t.Fatalf("importMsg.err = nil, want error")
		}
	})

	t.Run("too large file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "zone.json")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Truncate(maxImportFileSize + 1); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		_ = f.Close()

		cmd := importZoneCmd(&firewalld.Client{}, "public", path)
		msg := cmd()
		got, ok := msg.(importMsg)
		if !ok {
			t.Fatalf("msg type = %T, want importMsg", msg)
		}
		if got.err == nil {
			t.Fatalf("importMsg.err = nil, want error")
		}
		if !strings.Contains(got.err.Error(), "file too large") {
			t.Fatalf("importMsg.err = %v, want file too large", got.err)
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "zone.txt")
		if err := os.WriteFile(path, []byte("noop"), 0o644); err != nil {
			t.Fatal(err)
		}

		cmd := importZoneCmd(&firewalld.Client{}, "public", path)
		msg := cmd()
		got, ok := msg.(importMsg)
		if !ok {
			t.Fatalf("msg type = %T, want importMsg", msg)
		}
		if got.err == nil {
			t.Fatalf("importMsg.err = nil, want error")
		}
		if !strings.Contains(got.err.Error(), "unsupported import format") {
			t.Fatalf("importMsg.err = %v, want unsupported import format", got.err)
		}
	})
}

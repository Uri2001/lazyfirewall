//go:build linux
// +build linux

package firewalld

import (
	"errors"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestIsPermissionDenied(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "dbus access denied name",
			err:  &dbus.Error{Name: "org.freedesktop.DBus.Error.AccessDenied"},
			want: true,
		},
		{
			name: "dbus firewall not authorized name",
			err:  &dbus.Error{Name: "org.fedoraproject.FirewallD1.NotAuthorized"},
			want: true,
		},
		{
			name: "message contains permission denied",
			err:  errors.New("permission denied by polkit"),
			want: true,
		},
		{
			name: "message contains not authorized",
			err:  errors.New("not authorized"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("temporary network issue"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPermissionDenied(tt.err)
			if got != tt.want {
				t.Fatalf("isPermissionDenied(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAPIVersionString(t *testing.T) {
	tests := []struct {
		version APIVersion
		want    string
	}{
		{version: APIv1, want: "v1 (firewalld 0.x)"},
		{version: APIv2, want: "v2 (firewalld 1.x+)"},
		{version: APIUnknown, want: "unknown"},
		{version: APIVersion(99), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.want {
				t.Fatalf("APIVersion(%d).String() = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestNextDBusDelay(t *testing.T) {
	now := time.Unix(1000, 0)

	tests := []struct {
		name         string
		lastCall     time.Time
		wantPositive bool
		wantZero     bool
	}{
		{
			name:     "no previous call",
			lastCall: time.Time{},
			wantZero: true,
		},
		{
			name:         "too soon",
			lastCall:     now.Add(-(dbusMinCallGap - time.Millisecond)),
			wantPositive: true,
		},
		{
			name:     "after gap",
			lastCall: now.Add(-(dbusMinCallGap + time.Millisecond)),
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{lastDBusCall: tt.lastCall}
			got := c.nextDBusDelay(now)
			if tt.wantZero && got != 0 {
				t.Fatalf("nextDBusDelay() = %v, want 0", got)
			}
			if tt.wantPositive && got <= 0 {
				t.Fatalf("nextDBusDelay() = %v, want positive delay", got)
			}
		})
	}
}

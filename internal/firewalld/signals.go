//go:build linux
// +build linux

package firewalld

import (
	"fmt"
	"log/slog"

	"github.com/godbus/dbus/v5"
)

type SignalEvent struct {
	Name string
	Zone string
}

func (c *Client) SubscribeSignals() (<-chan SignalEvent, func(), error) {
	if c.conn == nil {
		return nil, nil, fmt.Errorf("dbus connection not initialized")
	}

	rule := "type='signal',sender='" + dbusInterface + "'"
	slog.Debug("dbus add match", "rule", rule)
	if call := c.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule); call.Err != nil {
		return nil, nil, fmt.Errorf("dbus add match: %w", call.Err)
	}

	raw := make(chan *dbus.Signal, 16)
	out := make(chan SignalEvent, 16)
	done := make(chan struct{})
	c.conn.Signal(raw)

	go func() {
		defer close(out)
		for {
			select {
			case sig := <-raw:
				if sig == nil {
					return
				}
				event := SignalEvent{Name: sig.Name}
				if len(sig.Body) > 0 {
					if zone, ok := sig.Body[0].(string); ok {
						event.Zone = zone
					}
				}
				out <- event
			case <-done:
				return
			}
		}
	}()

	cancel := func() {
		close(done)
		c.conn.RemoveSignal(raw)
		slog.Debug("dbus remove match", "rule", rule)
		_ = c.conn.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, rule).Err
	}

	return out, cancel, nil
}

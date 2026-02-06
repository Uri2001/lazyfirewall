//go:build linux
// +build linux

package firewalld

import (
	"sync/atomic"
	"testing"
)

func TestIdempotentCancel(t *testing.T) {
	var called int32
	cancel := idempotentCancel(func() {
		atomic.AddInt32(&called, 1)
	})

	cancel()
	cancel()
	cancel()

	if got := atomic.LoadInt32(&called); got != 1 {
		t.Fatalf("cancel callback called %d times, want 1", got)
	}
}

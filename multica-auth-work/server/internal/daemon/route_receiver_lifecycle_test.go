package daemon

import (
	"net"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/observability/e2e"
)

func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

// The route receiver must bind SYNCHRONOUSLY: a second bind on the same address
// returns an error immediately (not asynchronously after tasks can admit), and
// after Shutdown the address is free to rebind.
func TestRouteReceiverBindConflictAndRebind(t *testing.T) {
	rec := e2e.NewRecorder(nil)
	addr := freeLoopbackAddr(t)

	srv1, err := startRouteTelemetryReceiverOn(rec, nil, addr)
	if err != nil {
		t.Fatalf("first bind must succeed: %v", err)
	}

	// Synchronous conflict on the same address.
	if _, err2 := startRouteTelemetryReceiverOn(rec, nil, addr); err2 == nil {
		t.Fatal("second bind on the same address must fail synchronously")
	}

	// Shutdown frees the address; rebind succeeds (retry briefly for TIME_WAIT).
	shutdownRouteTelemetryReceiver(srv1)
	var srv2 interface{ Close() error }
	var rebindErr error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s, e := startRouteTelemetryReceiverOn(rec, nil, addr)
		if e == nil {
			srv2 = s
			rebindErr = nil
			break
		}
		rebindErr = e
		time.Sleep(50 * time.Millisecond)
	}
	if srv2 == nil {
		t.Fatalf("rebind after Shutdown must succeed: %v", rebindErr)
	}
	_ = srv2.Close()
}

func TestRouteReceiverNilRecorderFailsClosed(t *testing.T) {
	if _, err := startRouteTelemetryReceiverOn(nil, nil, "127.0.0.1:0"); err == nil {
		t.Fatal("nil recorder must fail closed")
	}
}

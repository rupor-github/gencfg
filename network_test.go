package gencfg

import (
	"net"
	"strconv"
	"testing"
)

func TestGetFreePort(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("getFreePort() returned error: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("getFreePort() returned invalid port: %d", port)
	}
}

func TestGetFreePortReturnsDistinctPorts(t *testing.T) {
	seen := make(map[int]struct{})
	for range 100 {
		port, err := getFreePort()
		if err != nil {
			t.Fatalf("getFreePort() returned error: %v", err)
		}
		seen[port] = struct{}{}
	}
	// The OS should generally give us multiple distinct ports over 100 calls.
	// Allow some duplicates, but not all the same.
	if len(seen) < 2 {
		t.Fatalf("getFreePort() returned only %d distinct port(s) over 100 calls", len(seen))
	}
}

func TestReservePort(t *testing.T) {
	// Use a high port unlikely to conflict with actual allocated ports.
	const testPort = 59123

	// Clean up portMap entry after test to avoid polluting global state.
	defer func() {
		guard.Lock()
		delete(portMap, testPort)
		guard.Unlock()
	}()

	if !reservePort(testPort) {
		t.Fatalf("reservePort(%d) returned false on first call", testPort)
	}
	if reservePort(testPort) {
		t.Fatalf("reservePort(%d) returned true on duplicate call", testPort)
	}
}

func TestFreeLocalPort(t *testing.T) {
	port, err := freeLocalPort()
	if err != nil {
		t.Fatalf("freeLocalPort() returned error: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("freeLocalPort() returned invalid port: %d", port)
	}
}

func TestFreeLocalPortIsReserved(t *testing.T) {
	port, err := freeLocalPort()
	if err != nil {
		t.Fatalf("freeLocalPort() returned error: %v", err)
	}
	// The port should already be in portMap, so reservePort should return false.
	if reservePort(port) {
		t.Fatalf("freeLocalPort() returned port %d that was not reserved in portMap", port)
	}
}

func TestFreeLocalPortIsListenable(t *testing.T) {
	port, err := freeLocalPort()
	if err != nil {
		t.Fatalf("freeLocalPort() returned error: %v", err)
	}
	// Verify we can actually listen on the returned port.
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("could not listen on port %d returned by freeLocalPort(): %v", port, err)
	}
	ln.Close()
}

func TestFreeLocalPortConcurrent(t *testing.T) {
	const goroutines = 20
	results := make(chan int, goroutines)
	errs := make(chan error, goroutines)

	for range goroutines {
		go func() {
			port, err := freeLocalPort()
			if err != nil {
				errs <- err
				return
			}
			results <- port
		}()
	}

	seen := make(map[int]struct{})
	for range goroutines {
		select {
		case err := <-errs:
			t.Fatalf("freeLocalPort() returned error in goroutine: %v", err)
		case port := <-results:
			if _, dup := seen[port]; dup {
				t.Fatalf("freeLocalPort() returned duplicate port %d across concurrent calls", port)
			}
			seen[port] = struct{}{}
		}
	}
}

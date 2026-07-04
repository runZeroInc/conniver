package main

import (
	"testing"
	"time"

	"github.com/runZeroInc/conniver"
)

// TestConniverPerHopReset checks that a hop never reads stats left from a
// prior hop: reset clears the recorded conn and the wait only observes the
// callback fired after the reset.
func TestConniverPerHopReset(t *testing.T) {
	// hop A records a conn.
	a := &conniver.Conn{}
	setConniver(a)
	if getConniver() != a {
		t.Fatalf("hop A: want recorded conn, got %v", getConniver())
	}

	// hop B starts: reset must drop hop A's stats.
	setConniver(nil)
	if getConniver() != nil {
		t.Fatalf("hop B start: want nil, got stale %v", getConniver())
	}

	// with no callback yet, wait returns via timeout and conn stays nil.
	start := time.Now()
	waitConniver(50 * time.Millisecond)
	if getConniver() != nil {
		t.Fatalf("hop B before callback: want nil, got %v", getConniver())
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Fatalf("wait returned too early, ready channel not honored")
	}

	// hop B's callback fires: wait now unblocks promptly with hop B's conn.
	setConniver(nil)
	b := &conniver.Conn{}
	go func() {
		time.Sleep(10 * time.Millisecond)
		setConniver(b)
	}()
	start = time.Now()
	waitConniver(2 * time.Second)
	if time.Since(start) > time.Second {
		t.Fatalf("wait did not unblock on callback")
	}
	if getConniver() != b {
		t.Fatalf("hop B: want %v, got %v", b, getConniver())
	}
}

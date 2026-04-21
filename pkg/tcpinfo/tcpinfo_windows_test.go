//go:build windows

package tcpinfo

import (
	"testing"
	"time"
)

func TestRawInfoV1UnpackUsesMillisecondsForConnectedTime(t *testing.T) {
	got := (&RawInfoV1{ConnectionTimeMs: 3}).Unpack()
	if got.ConnectedTimeNS != 3*time.Millisecond {
		t.Fatalf("ConnectedTimeNS = %s, want %s", got.ConnectedTimeNS, 3*time.Millisecond)
	}
}

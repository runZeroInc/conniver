//go:build windows

package tcpinfo

import (
	"net"
	"testing"
	"time"
)

func TestRawInfoV1UnpackUsesMillisecondsForConnectedTime(t *testing.T) {
	got := (&RawInfoV1{ConnectionTimeMs: 3}).Unpack()
	if got.ConnectedTimeNS != 3*time.Millisecond {
		t.Fatalf("ConnectedTimeNS = %s, want %s", got.ConnectedTimeNS, 3*time.Millisecond)
	}
}

func TestGetTCPInfo_LiveSocket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}

	conn, err := net.DialTimeout("tcp", "1.1.1.1:443", 5*time.Second)
	if err != nil {
		t.Skipf("skipping: cannot reach 1.1.1.1:443: %v", err)
	}
	defer conn.Close()

	tcpConn := conn.(*net.TCPConn)
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn: %v", err)
	}

	var sysInfo *SysInfo
	var infoErr error
	if err := rawConn.Control(func(fd uintptr) {
		sysInfo, infoErr = GetTCPInfo(fd)
	}); err != nil {
		t.Fatalf("Control: %v", err)
	}
	if infoErr != nil {
		t.Fatalf("GetTCPInfo: %v", infoErr)
	}
	if sysInfo == nil {
		t.Fatal("GetTCPInfo returned nil SysInfo")
	}

	t.Logf("SysInfo: %+v", sysInfo)

	if sysInfo.StateName != "ESTABLISHED" {
		t.Errorf("StateName = %q, want ESTABLISHED", sysInfo.StateName)
	}
	if sysInfo.RTT == 0 {
		t.Error("RTT = 0, want > 0")
	}
	if sysInfo.MSS == 0 {
		t.Error("MSS = 0, want > 0")
	}

	// Verify ToInfo round-trips the key fields correctly.
	info := sysInfo.ToInfo()
	if info == nil {
		t.Fatal("ToInfo returned nil")
	}
	if info.TxMSS != uint64(sysInfo.MSS) {
		t.Errorf("Info.TxMSS = %d, want %d", info.TxMSS, sysInfo.MSS)
	}
	if info.Sys != sysInfo {
		t.Error("Info.Sys does not point to the original SysInfo")
	}
}

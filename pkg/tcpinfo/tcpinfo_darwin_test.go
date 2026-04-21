//go:build darwin

package tcpinfo

import (
	"net"
	"testing"
	"time"
)

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
	if sysInfo.SRTT == 0 {
		t.Error("SRTT = 0, want > 0")
	}
	if sysInfo.MaxSeg == 0 {
		t.Error("MaxSeg = 0, want > 0")
	}

	// Verify ToInfo round-trips the key fields correctly.
	info := sysInfo.ToInfo()
	if info == nil {
		t.Fatal("ToInfo returned nil")
	}
	if info.RTT != sysInfo.SRTT {
		t.Errorf("Info.RTT = %v, want %v (SRTT)", info.RTT, sysInfo.SRTT)
	}
	if info.Sys != sysInfo {
		t.Error("Info.Sys does not point to the original SysInfo")
	}
}

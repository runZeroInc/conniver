package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/runZeroInc/conniver/pkg/tcpinfo"
)

func main() {
	conn, err := net.Dial("tcp", "google.com:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	sysConn, ok := conn.(*net.TCPConn)
	if !ok {
		panic("not a TCP connection")
	}

	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return
	}

	var sysInfo *tcpinfo.SysInfo
	if err := rawConn.Control(func(fd uintptr) {
		// Pass the `fd` to GetTCPInfo here
		sysInfo, err = tcpinfo.GetTCPInfo(fd)
	}); err != nil {
		return
	}

	jb, _ := json.MarshalIndent(sysInfo, "", "  ")
	fmt.Printf("%s\n", string(jb))
}

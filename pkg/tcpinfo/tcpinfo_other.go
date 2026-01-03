//go:build !darwin && !linux && !windows
// +build !darwin,!linux,!windows

package tcpinfo

import (
	"fmt"
	"runtime"
)

type SysInfo struct {
	// Empty for unsupported platforms
}

func (s *SysInfo) ToInfo() *Info {
	return &Info{}
}

func (s *SysInfo) Warnings() []string {
	return nil
}

func GetTCPInfo(fd int) (*SysInfo, error) {
	return nil, fmt.Errorf("%s is unsupported", runtime.GOOS)
}

func Supported() bool {
	return false
}

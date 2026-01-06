//go:build !(linux || darwin || windows)

package tcpinfo

import (
	"encoding/json"
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

func (s *SysInfo) ToMap() map[string]any {
	return map[string]any{}
}

func (s *SysInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ToMap())
}

func GetTCPInfo(fd uintptr) (*SysInfo, error) {
	return nil, fmt.Errorf("%s is unsupported", runtime.GOOS)
}

func Supported() bool {
	return false
}

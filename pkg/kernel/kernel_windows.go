//go:build windows

package kernel

import (
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// VersionInfo holds information about the kernel.
type VersionInfo struct {
	kvi   string // Version of the kernel (e.g. 6.1.7601.17592 -> 6)
	major int    // Major part of the kernel version (e.g. 6.1.7601.17592 -> 1)
	minor int    // Minor part of the kernel version (e.g. 6.1.7601.17592 -> 7601)
	build int    // Build number of the kernel version (e.g. 6.1.7601.17592 -> 17592)
}

func (k *VersionInfo) String() string {
	return fmt.Sprintf("%d.%d %d (%s)", k.major, k.minor, k.build, k.kvi)
}

// GetKernelVersion gets the current kernel version.
func GetKernelVersion() (*VersionInfo, error) {
	KVI := &VersionInfo{"Unknown", 0, 0, 0}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return KVI, err
	}
	defer k.Close()

	blex, _, err := k.GetStringValue("BuildLabEx")
	if err != nil {
		return KVI, err
	}
	KVI.kvi = blex

	// RtlGetVersion ignores the manifest shim that caps GetVersion at 6.2.9200
	// for unmanifested processes, so it reports the real version on Win 8.1+.
	ver := windows.RtlGetVersion()

	KVI.major = int(ver.MajorVersion)
	KVI.minor = int(ver.MinorVersion)
	KVI.build = int(ver.BuildNumber)

	return KVI, nil
}

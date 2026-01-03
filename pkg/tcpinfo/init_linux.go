//go:build linux
// +build linux

/**
 * Copyright (c) 2022, Xerra Earth Observation Institute.
 * Copyright (c) 2025, Simeon Miteff.
 *
 * See LICENSE.TXT in the root directory of this source tree.
 */

package tcpinfo

import (
	"github.com/runZeroInc/conniver/pkg/kernel"
)

var (
	linuxKernelVersion *kernel.VersionInfo
	sizeOfRawTCPInfo   int
)

type VersionedStructSize struct {
	Version kernel.VersionInfo
	Size    int
	Flag    *bool
}

var (
	kernelVersionIsAtLeast_2_6_2 = false
	kernelVersionIsAtLeast_3_15  = false
	kernelVersionIsAtLeast_4_1   = false
	kernelVersionIsAtLeast_4_2   = false
	kernelVersionIsAtLeast_4_6   = false
	kernelVersionIsAtLeast_4_9   = false
	kernelVersionIsAtLeast_4_10  = false
	kernelVersionIsAtLeast_4_18  = false
	kernelVersionIsAtLeast_4_19  = false
	kernelVersionIsAtLeast_5_4   = false
	kernelVersionIsAtLeast_5_5   = false
	kernelVersionIsAtLeast_6_2   = false
	kernelVersionIsAtLeast_6_7   = false
)

var tcpInfoSizes = []VersionedStructSize{
	{Version: kernel.VersionInfo{Kernel: 2, Major: 6, Minor: 2}, Size: 104, Flag: &kernelVersionIsAtLeast_2_6_2},
	{Version: kernel.VersionInfo{Kernel: 3, Major: 15, Minor: 0}, Size: 120, Flag: &kernelVersionIsAtLeast_3_15},
	{Version: kernel.VersionInfo{Kernel: 4, Major: 1, Minor: 0}, Size: 136, Flag: &kernelVersionIsAtLeast_4_1},
	{Version: kernel.VersionInfo{Kernel: 4, Major: 2, Minor: 0}, Size: 144, Flag: &kernelVersionIsAtLeast_4_2},
	{Version: kernel.VersionInfo{Kernel: 4, Major: 6, Minor: 0}, Size: 160, Flag: &kernelVersionIsAtLeast_4_6},
	{Version: kernel.VersionInfo{Kernel: 4, Major: 9, Minor: 0}, Size: 148, Flag: &kernelVersionIsAtLeast_4_9},
	{Version: kernel.VersionInfo{Kernel: 4, Major: 10, Minor: 0}, Size: 192, Flag: &kernelVersionIsAtLeast_4_10},
	{Version: kernel.VersionInfo{Kernel: 4, Major: 18, Minor: 0}, Size: 200, Flag: &kernelVersionIsAtLeast_4_18},
	{Version: kernel.VersionInfo{Kernel: 4, Major: 19, Minor: 0}, Size: 224, Flag: &kernelVersionIsAtLeast_4_19},
	{Version: kernel.VersionInfo{Kernel: 5, Major: 4, Minor: 0}, Size: 232, Flag: &kernelVersionIsAtLeast_5_4},
	{Version: kernel.VersionInfo{Kernel: 5, Major: 5, Minor: 0}, Size: 232, Flag: &kernelVersionIsAtLeast_5_5},
	{Version: kernel.VersionInfo{Kernel: 6, Major: 2, Minor: 0}, Size: 240, Flag: &kernelVersionIsAtLeast_6_2},
	{Version: kernel.VersionInfo{Kernel: 6, Major: 7, Minor: 0}, Size: 248, Flag: &kernelVersionIsAtLeast_6_7},
}

func init() {
	adaptToKernelVersion()
}

func adaptToKernelVersion() {
	linuxKernelVersion, err := kernel.GetKernelVersion()
	if err != nil {
		linuxKernelVersion = &kernel.VersionInfo{Kernel: 2, Major: 6, Minor: 2} // Fallback to very old kernel version
	}

	for i := len(tcpInfoSizes) - 1; i >= 0; i-- {
		if kernel.CompareKernelVersion(*linuxKernelVersion, tcpInfoSizes[i].Version) >= 0 {
			sizeOfRawTCPInfo = tcpInfoSizes[i].Size

			for j := i; j >= 0; j-- {
				*tcpInfoSizes[j].Flag = true
			}

			return
		} else {
			*tcpInfoSizes[i].Flag = false // Needed if we manually override linuxKernelVersion in unit tests
		}
	}
}

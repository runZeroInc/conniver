//go:build linux

/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

package linux

import (
	"fmt"

	"github.com/docker/docker/pkg/parsers/kernel"
)

// A future version of this library could support older kernels
const minKernel = 5
const minKernelMajor = 4
const minKernelMinor = 0

var linuxKernelVersion *kernel.VersionInfo

func init() {
	var err error
	if linuxKernelVersion, err = kernel.GetKernelVersion(); err != nil {
		panic(fmt.Errorf("error getting kernel version: %s", err))
	}

	if !CheckKernelVersion(minKernel, minKernelMajor, minKernelMinor) {
		panic(fmt.Sprintf("Linux kernel is too old to use go-tcpinfo (want >= %d.%d.%d, got %d.%d.%d)",
			minKernel, minKernelMajor, minKernelMinor, linuxKernelVersion.Kernel, linuxKernelVersion.Major, linuxKernelVersion.Minor))
	}
}

func CheckKernelVersion(k, major, minor int) bool {
	if kernel.CompareKernelVersion(*linuxKernelVersion, kernel.VersionInfo{Kernel: k, Major: major, Minor: minor}) < 0 {
		return false
	}
	return true
}

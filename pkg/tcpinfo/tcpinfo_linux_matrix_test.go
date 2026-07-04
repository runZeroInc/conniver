//go:build linux

/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

package tcpinfo

import (
	"testing"
	"unsafe"

	"github.com/runZeroInc/conniver/pkg/kernel"
)

// These tests exist because the version-adaptation logic was, for a long time,
// only ever exercised at the host kernel's version (see the shadowing bug fixed
// alongside them). Once injection actually works, the size table and the
// per-field version gating in Unpack become independently testable across the
// whole matrix, rather than at a single incidental data point.

// TestSizeTableInvariants pins down two structural properties of tcpInfoSizes
// that hold for any append-only C struct:
//
//  1. Sizes are monotonically non-decreasing with kernel version. A struct that
//     only ever gains trailing fields can never shrink. (This alone catches the
//     historical v4.9=148 regression, which sat below v4.6=160.)
//  2. The newest row equals the actual size of the Go mirror struct. This ties
//     the top of the table to ground truth and catches drift whenever a field
//     is appended to RawTCPInfo without a matching table update.
func TestSizeTableInvariants(t *testing.T) {
	prev := -1
	var prevV kernel.VersionInfo
	for _, row := range tcpInfoSizes {
		if row.Size < prev {
			t.Errorf("non-monotonic size table: v%s=%d is smaller than v%s=%d; "+
				"an append-only struct cannot shrink across versions",
				row.Version.String(), row.Size, prevV.String(), prev)
		}
		prev = row.Size
		prevV = row.Version
	}

	want := int(unsafe.Sizeof(RawTCPInfo{}))
	last := tcpInfoSizes[len(tcpInfoSizes)-1]
	if last.Size != want {
		t.Errorf("newest size row v%s=%d does not match sizeof(RawTCPInfo)=%d; "+
			"table and struct have drifted apart",
			last.Version.String(), last.Size, want)
	}
}

// fieldIntroducedIn records the first kernel version in which each nullable
// tcp_info field became populated, taken from the commit annotations on the
// RawTCPInfo struct definition. It is the ground truth the Unpack version gating
// must agree with.
var fieldIntroducedIn = []struct {
	name    string
	version kernel.VersionInfo
	valid   func(*SysInfo) bool
}{
	{"PacingRate", kernel.VersionInfo{Kernel: 3, Major: 15}, func(s *SysInfo) bool { return s.PacingRate.Valid }},
	{"MaxPacingRate", kernel.VersionInfo{Kernel: 3, Major: 15}, func(s *SysInfo) bool { return s.MaxPacingRate.Valid }},
	{"BytesAcked", kernel.VersionInfo{Kernel: 4, Major: 1}, func(s *SysInfo) bool { return s.BytesAcked.Valid }},
	{"BytesReceived", kernel.VersionInfo{Kernel: 4, Major: 1}, func(s *SysInfo) bool { return s.BytesReceived.Valid }},
	{"SegsOut", kernel.VersionInfo{Kernel: 4, Major: 2}, func(s *SysInfo) bool { return s.SegsOut.Valid }},
	{"SegsIn", kernel.VersionInfo{Kernel: 4, Major: 2}, func(s *SysInfo) bool { return s.SegsIn.Valid }},
	{"NotSentBytes", kernel.VersionInfo{Kernel: 4, Major: 6}, func(s *SysInfo) bool { return s.NotSentBytes.Valid }},
	{"MinRTT", kernel.VersionInfo{Kernel: 4, Major: 6}, func(s *SysInfo) bool { return s.MinRTT.Valid }},
	{"DataSegsIn", kernel.VersionInfo{Kernel: 4, Major: 6}, func(s *SysInfo) bool { return s.DataSegsIn.Valid }},
	{"DataSegsOut", kernel.VersionInfo{Kernel: 4, Major: 6}, func(s *SysInfo) bool { return s.DataSegsOut.Valid }},
	{"DeliveryRateAppLimited", kernel.VersionInfo{Kernel: 4, Major: 9}, func(s *SysInfo) bool { return s.DeliveryRateAppLimited.Valid }},
	{"DeliveryRate", kernel.VersionInfo{Kernel: 4, Major: 9}, func(s *SysInfo) bool { return s.DeliveryRate.Valid }},
	{"BusyTime", kernel.VersionInfo{Kernel: 4, Major: 10}, func(s *SysInfo) bool { return s.BusyTime.Valid }},
	{"RxWindowLimited", kernel.VersionInfo{Kernel: 4, Major: 10}, func(s *SysInfo) bool { return s.RxWindowLimited.Valid }},
	{"TxBufferLimited", kernel.VersionInfo{Kernel: 4, Major: 10}, func(s *SysInfo) bool { return s.TxBufferLimited.Valid }},
	{"Delivered", kernel.VersionInfo{Kernel: 4, Major: 18}, func(s *SysInfo) bool { return s.Delivered.Valid }},
	{"DeliveredCE", kernel.VersionInfo{Kernel: 4, Major: 18}, func(s *SysInfo) bool { return s.DeliveredCE.Valid }},
	{"BytesSent", kernel.VersionInfo{Kernel: 4, Major: 19}, func(s *SysInfo) bool { return s.BytesSent.Valid }},
	{"BytesRetrans", kernel.VersionInfo{Kernel: 4, Major: 19}, func(s *SysInfo) bool { return s.BytesRetrans.Valid }},
	{"DSACKDups", kernel.VersionInfo{Kernel: 4, Major: 19}, func(s *SysInfo) bool { return s.DSACKDups.Valid }},
	{"ReordSeen", kernel.VersionInfo{Kernel: 4, Major: 19}, func(s *SysInfo) bool { return s.ReordSeen.Valid }},
	{"RxOutOfOrder", kernel.VersionInfo{Kernel: 5, Major: 4}, func(s *SysInfo) bool { return s.RxOutOfOrder.Valid }},
	{"TxWindow", kernel.VersionInfo{Kernel: 5, Major: 4}, func(s *SysInfo) bool { return s.TxWindow.Valid }},
	{"FastOpenClientFail", kernel.VersionInfo{Kernel: 5, Major: 5}, func(s *SysInfo) bool { return s.FastOpenClientFail.Valid }},
	{"RxWindow", kernel.VersionInfo{Kernel: 6, Major: 2}, func(s *SysInfo) bool { return s.RxWindow.Valid }},
	{"Rehash", kernel.VersionInfo{Kernel: 6, Major: 2}, func(s *SysInfo) bool { return s.Rehash.Valid }},
	{"TotalRTO", kernel.VersionInfo{Kernel: 6, Major: 7}, func(s *SysInfo) bool { return s.TotalRTO.Valid }},
	{"TotalRTORecoveries", kernel.VersionInfo{Kernel: 6, Major: 7}, func(s *SysInfo) bool { return s.TotalRTORecoveries.Valid }},
	{"TotalRTOTime", kernel.VersionInfo{Kernel: 6, Major: 7}, func(s *SysInfo) bool { return s.TotalRTOTime.Valid }},
}

// TestVersionFieldGating asserts that, for a spread of kernel versions
// (including the boundary cases that previously hid bugs), Unpack marks each
// field Valid if and only if that field's introduction version is <= the pinned
// kernel. The 6.6.0 case is the direct regression guard for the RTO fields that
// were mis-gated under _6_2: on 6.6 they must be absent, on 6.7 present.
func TestVersionFieldGating(t *testing.T) {
	pins := []kernel.VersionInfo{
		{Kernel: 2, Major: 6, Minor: 2},
		{Kernel: 4, Major: 8, Minor: 0}, // just below delivery_rate (4.9)
		{Kernel: 4, Major: 9, Minor: 0}, // delivery_rate present, size must be 168
		{Kernel: 5, Major: 4, Minor: 0},
		{Kernel: 6, Major: 2, Minor: 0}, // rcv_wnd/rehash present, RTO fields absent
		{Kernel: 6, Major: 6, Minor: 0}, // still below 6.7: RTO fields MUST be absent
		{Kernel: 6, Major: 7, Minor: 0}, // RTO fields present
	}

	for _, pin := range pins {
		pin := pin
		t.Run(pin.String(), func(t *testing.T) {
			linuxKernelVersion = &pin
			adaptToKernelVersion()
			got := (&RawTCPInfo{}).Unpack()
			for _, f := range fieldIntroducedIn {
				wantValid := kernel.CompareKernelVersion(pin, f.version) >= 0
				if f.valid(got) != wantValid {
					t.Errorf("at kernel %s: field %s Valid=%v, want %v (introduced in %s)",
						pin.String(), f.name, f.valid(got), wantValid, f.version.String())
				}
			}
		})
	}
}

// TestInjectionHolds is the direct regression test for the shadowing bug: a
// pinned version must actually determine sizeOfRawTCPInfo and the flags,
// independent of the host kernel that runs the test.
func TestInjectionHolds(t *testing.T) {
	linuxKernelVersion = &kernel.VersionInfo{Kernel: 2, Major: 6, Minor: 2}
	adaptToKernelVersion()
	if sizeOfRawTCPInfo != 104 {
		t.Errorf("pinned 2.6.2 but sizeOfRawTCPInfo=%d, want 104 (injection discarded?)", sizeOfRawTCPInfo)
	}
	if kernelVersionIsAtLeast_3_15 {
		t.Error("pinned 2.6.2 but kernelVersionIsAtLeast_3_15 is true (host kernel leaked in)")
	}

	linuxKernelVersion = &kernel.VersionInfo{Kernel: 6, Major: 7, Minor: 0}

	adaptToKernelVersion()
	if !kernelVersionIsAtLeast_6_7 {
		t.Error("pinned 6.7.0 but kernelVersionIsAtLeast_6_7 is false")
	}
}

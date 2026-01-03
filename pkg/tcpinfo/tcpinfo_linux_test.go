//go:build linux
// +build linux

/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

package tcpinfo

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/runZeroInc/conniver/pkg/kernel"
)

const (
	minKernel      int = 5
	minKernelMajor int = 5
	minKernelMinor int = 0
)

func TestRawTCPInfo_Unpack(t *testing.T) {
	type fields struct {
		kernel                 kernel.VersionInfo
		SendWScale             uint8
		RecvWScale             uint8
		DeliveryRateAppLimited NullableBool
		FastOpenClientFail     NullableUint8
	}

	baseDesire := SysInfo{
		DeliveryRateAppLimited: NullableBool{Valid: true},
		FastOpenClientFail:     NullableUint8{Valid: true},
		PacingRate:             NullableUint64{Valid: true},
		MaxPacingRate:          NullableUint64{Valid: true},
		BytesAcked:             NullableUint64{Valid: true},
		BytesReceived:          NullableUint64{Valid: true},
		SegsOut:                NullableUint32{Valid: true},
		SegsIn:                 NullableUint32{Valid: true},
		NotSentBytes:           NullableUint32{Valid: true},
		MinRTT:                 NullableDuration{Valid: true},
		DataSegsIn:             NullableUint32{Valid: true},
		DataSegsOut:            NullableUint32{Valid: true},
		DeliveryRate:           NullableUint64{Valid: true},
		BusyTime:               NullableUint64{Valid: true},
		RwndLimited:            NullableUint64{Valid: true},
		SndbufLimited:          NullableUint64{Valid: true},
		Delivered:              NullableUint32{Valid: true},
		DeliveredCE:            NullableUint32{Valid: true},
		BytesSent:              NullableUint64{Valid: true},
		BytesRetrans:           NullableUint64{Valid: true},
		DSACKDups:              NullableUint32{Valid: true},
		ReordSeen:              NullableUint32{Valid: true},
		RcvOOOPacket:           NullableUint32{Valid: true},
		SendWindow:             NullableUint32{Valid: true},
		RecvWindow:             NullableUint32{Valid: true},
		Rehash:                 NullableUint32{Valid: true},
		TotalRTO:               NullableUint16{Valid: true},
		TotalRTORecoveries:     NullableUint16{Valid: true},
		TotalRTOTime:           NullableUint32{Valid: true},
	}

	wantDeliveryRateAppLimited := baseDesire
	wantDeliveryRateAppLimited.DeliveryRateAppLimited.Value = true

	wanFastOpenClientFail0 := baseDesire

	wanFastOpenClientFail1 := baseDesire
	wanFastOpenClientFail1.FastOpenClientFail.Value = 1

	wanFastOpenClientFail2 := baseDesire
	wanFastOpenClientFail2.FastOpenClientFail.Value = 2

	wantSndWScale1 := baseDesire
	wantSndWScale1.SndWScale = 1

	wantRcvWScale1 := baseDesire
	wantRcvWScale1.RcvWScale = 1

	wantSndWScaleF := baseDesire
	wantSndWScaleF.SndWScale = 0xf

	wantRcvWScaleF := baseDesire
	wantRcvWScaleF.RcvWScale = 0xf

	tests := []struct {
		name   string
		fields fields
		want   *SysInfo
	}{
		{
			name: "zeros",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0,
				RecvWScale:             0,
				DeliveryRateAppLimited: NullableBool{},
				FastOpenClientFail:     NullableUint8{},
			},
			want: &baseDesire,
		},
		{
			name: "SndWScale1",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             1,
				RecvWScale:             0,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: false},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 0},
			},
			want: &wantSndWScale1,
		},
		{
			name: "RcvWScale1",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0,
				RecvWScale:             1,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: false},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 0},
			},
			want: &wantRcvWScale1,
		},
		{
			name: "SndWScaleF",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0xf,
				RecvWScale:             0,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: false},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 0},
			},
			want: &wantSndWScaleF,
		},
		{
			name: "RcvWScaleF",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0,
				RecvWScale:             0xf,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: false},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 0},
			},
			want: &wantRcvWScaleF,
		},
		{
			name: "DeliveryRateAppLimited",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0,
				RecvWScale:             0,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: true},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 0},
			},
			want: &wantDeliveryRateAppLimited,
		},
		{
			name: "FastOpenClientFail0",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0,
				RecvWScale:             0,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: false},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 0},
			},
			want: &wanFastOpenClientFail0,
		},
		{
			name: "FastOpenClientFail0",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0,
				RecvWScale:             0,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: false},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 1},
			},
			want: &wanFastOpenClientFail1,
		},
		{
			name: "FastOpenClientFail2",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SendWScale:             0,
				RecvWScale:             0,
				DeliveryRateAppLimited: NullableBool{Valid: true, Value: false},
				FastOpenClientFail:     NullableUint8{Valid: true, Value: 2},
			},
			want: &wanFastOpenClientFail2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw RawInfo
			linuxKernelVersion = &tt.fields.kernel
			adaptToKernelVersion()
			if got := raw.Unpack(); !reflect.DeepEqual(got, tt.want) {
				for n, s := range tcpInfoSizes {
					fmt.Printf("%d tcpIntoSize = %#v | %v\n", n, s.Version, *s.Flag)
				}

				t.Errorf("For %s Unpack():\n\t got = %#v\n\twant = %#v", tt.name, got, tt.want)
			}
		})
	}
}

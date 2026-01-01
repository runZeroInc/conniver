/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

package linux

import (
	"reflect"
	"testing"

	"github.com/docker/docker/pkg/parsers/kernel"
)

func TestRawTCPInfo_Unpack(t *testing.T) {
	type fields struct {
		kernel                 kernel.VersionInfo
		SndWScale              uint8
		RcvWScale              uint8
		DeliveryRateAppLimited bool
		FastOpenClientFail     uint8
	}
	tests := []struct {
		name   string
		fields fields
		want   *TCPInfo
	}{
		{
			name: "zeros",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SndWScale:              0,
				RcvWScale:              0,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     0,
			},
			want: &TCPInfo{},
		},
		{
			name: "SndWScale1",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SndWScale:              1,
				RcvWScale:              0,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     0,
			},
			want: &TCPInfo{SndWScale: 1},
		},
		{
			name: "RcvWScale1",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SndWScale:              0,
				RcvWScale:              1,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     0,
			},
			want: &TCPInfo{RcvWScale: 1},
		},
		{
			name: "SndWScaleF",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SndWScale:              0xf,
				RcvWScale:              0,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     0,
			},
			want: &TCPInfo{SndWScale: 0xf},
		},
		{
			name: "RcvWScaleF",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SndWScale:              0,
				RcvWScale:              0xf,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     0,
			},
			want: &TCPInfo{RcvWScale: 0xf},
		},
		{
			name: "DeliveryRateAppLimited",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: minKernel, Major: minKernelMajor, Minor: minKernelMinor},
				SndWScale:              0,
				RcvWScale:              0,
				DeliveryRateAppLimited: true,
				FastOpenClientFail:     0,
			},
			want: &TCPInfo{DeliveryRateAppLimited: true},
		},
		{
			name: "FastOpenClientFail0",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: 5, Major: 5, Minor: 0},
				SndWScale:              0,
				RcvWScale:              0,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     0,
			},
			want: &TCPInfo{FastOpenClientFail: NullableUint8{
				Valid: true,
				Value: 0,
			}},
		},
		{
			name: "FastOpenClientFail1",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: 5, Major: 5, Minor: 0},
				SndWScale:              0,
				RcvWScale:              0,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     1,
			},
			want: &TCPInfo{FastOpenClientFail: NullableUint8{
				Valid: true,
				Value: 1,
			}},
		},
		{
			name: "FastOpenClientFail2",
			fields: fields{
				kernel:                 kernel.VersionInfo{Kernel: 5, Major: 5, Minor: 0},
				SndWScale:              0,
				RcvWScale:              0,
				DeliveryRateAppLimited: false,
				FastOpenClientFail:     2,
			},
			want: &TCPInfo{FastOpenClientFail: NullableUint8{
				Valid: true,
				Value: 2,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw RawTCPInfo
			raw.MockSetFields(
				tt.fields.SndWScale,
				tt.fields.RcvWScale,
				tt.fields.DeliveryRateAppLimited,
				tt.fields.FastOpenClientFail,
			)
			linuxKernelVersion = &tt.fields.kernel
			if got := raw.Unpack(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unpack() = %v, want %v", got, tt.want)
			}
		})
	}
}

//go:build linux

/**
 * Copyright (c) 2022, Xerra Earth Observation Institute.
 *
 * Portions are derived from of Linux's tcp.h, used under the syscall exception
 * (see https://spdx.org/licenses/Linux-syscall-note.html).
 *
 * See LICENSE.TXT in the root directory of this source tree.
 */

package linux

import (
	"syscall"
	"unsafe"
)

// RawTCPInfo has identical memory layout to Linux kernel tcp_info struct (current as of kernel 5.17.0).
// bitfield0 and bitfield1 have been added to capture the 4 packed fields. Note that bitfield1 would still
// have had the same location before tcpi_delivery_rate_app_limited and tcpi_fastopen_client_fail were added
// (in v4.9.0 and v5.5.0 respectively) because of alignment rules, so they didn't increase the length or
// shift the offsets of subsequent variables.
type RawTCPInfo struct { //struct tcp_info {          																		// unless noted below, struct fields have been around since at least (1da177e4c3f41524e886b7f1b8a0c1fc7321cac2) v2.6.12-rc2^0
	state           uint8  //__u8	tcpi_state;
	ca_state        uint8  //__u8	tcpi_ca_state;
	retransmits     uint8  //__u8	tcpi_retransmits;
	probes          uint8  //__u8	tcpi_probes;
	backoff         uint8  //__u8	tcpi_backoff;
	options         uint8  //__u8	tcpi_options;
	bitfield0       uint8  //__u8	tcpi_snd_wscale : 4, tcpi_rcv_wscale : 4;
	bitfield1       uint8  //__u8	tcpi_delivery_rate_app_limited:1, tcpi_fastopen_client_fail:2; 							// added via commits eb8329e0a04db0061f714f033b4454326ba147f4 (v4.9-rc1~127^2~120^2~7) and 480274787d7e3458bc5a7cfbbbe07033984ad711 (v5.5-rc1~174^2~318) respectively
	rto             uint32 //__u32	tcpi_rto;
	ato             uint32 //__u32	tcpi_ato;
	snd_mss         uint32 //__u32	tcpi_snd_mss;
	rcv_mss         uint32 //__u32	tcpi_rcv_mss;
	unacked         uint32 //__u32	tcpi_unacked;
	sacked          uint32 //__u32	tcpi_sacked;
	lost            uint32 //__u32	tcpi_lost;
	retrans         uint32 //__u32	tcpi_retrans;
	fackets         uint32 //__u32	tcpi_fackets;
	last_data_sent  uint32 //__u32	tcpi_last_data_sent;
	last_ack_sent   uint32 //__u32	tcpi_last_ack_sent;     /* Not remembered, sorry. */
	last_data_recv  uint32 //__u32	tcpi_last_data_recv;
	last_ack_recv   uint32 //__u32	tcpi_last_ack_recv;
	pmtu            uint32 //__u32	tcpi_pmtu;
	rcv_ssthresh    uint32 //__u32	tcpi_rcv_ssthresh;
	rtt             uint32 //__u32	tcpi_rtt;
	rttvar          uint32 //__u32	tcpi_rttvar;
	snd_ssthresh    uint32 //__u32	tcpi_snd_ssthresh;
	snd_cwnd        uint32 //__u32	tcpi_snd_cwnd;
	advmss          uint32 //__u32	tcpi_advmss;
	reordering      uint32 //__u32	tcpi_reordering;
	rcv_rtt         uint32 //__u32	tcpi_rcv_rtt;
	rcv_space       uint32 //__u32	tcpi_rcv_space;
	total_retrans   uint32 //__u32	tcpi_total_retrans;
	pacing_rate     uint64 //__u64	tcpi_pacing_rate; 																		// added via commit 977cb0ecf82eb6d15562573c31edebf90db35163 (v3.15-rc1~113^2~349)
	max_pacing_rate uint64 //__u64	tcpi_max_pacing_rate; 																	// added via commit 977cb0ecf82eb6d15562573c31edebf90db35163 (v3.15-rc1~113^2~349)
	bytes_acked     uint64 //__u64	tcpi_bytes_acked;    /* RFC4898 tcpEStatsAppHCThruOctetsAcked */ 						// added via commit 2efd055c53c06b7e89c167c98069bab9afce7e59 (v4.2-rc1~130^2~238)
	bytes_received  uint64 //__u64	tcpi_bytes_received; /* RFC4898 tcpEStatsAppHCThruOctetsReceived */ 					// added via commit bdd1f9edacb5f5835d1e6276571bbbe5b88ded48 (v4.1-rc4~26^2~34^2~21)
	segs_out        uint32 //__u32	tcpi_segs_out;	     /* RFC4898 tcpEStatsPerfSegsOut */ 								// added via commit 2efd055c53c06b7e89c167c98069bab9afce7e59 (v4.2-rc1~130^2~238)
	segs_in         uint32 //__u32	tcpi_segs_in;	     /* RFC4898 tcpEStatsPerfSegsIn */ 									// added via commit 2efd055c53c06b7e89c167c98069bab9afce7e59 (v4.2-rc1~130^2~238)
	notsent_bytes   uint32 //__u32	tcpi_notsent_bytes; 																	// added via commit cd9b266095f422267bddbec88f9098b48ea548fc (v4.6-rc1~91^2~262)
	min_rtt         uint32 //__u32	tcpi_min_rtt; 																			// added via commit cd9b266095f422267bddbec88f9098b48ea548fc (v4.6-rc1~91^2~262)
	data_segs_in    uint32 //__u32	tcpi_data_segs_in;	/* RFC4898 tcpEStatsDataSegsIn */ 									// added via commit a44d6eacdaf56f74fad699af7f4925a5f5ac0e7f (v4.6-rc1~91^2~51)
	data_segs_out   uint32 //__u32	tcpi_data_segs_out;	/* RFC4898 tcpEStatsDataSegsOut */ 									// added via commit a44d6eacdaf56f74fad699af7f4925a5f5ac0e7f (v4.6-rc1~91^2~51)
	delivery_rate   uint64 //__u64  tcpi_delivery_rate; 																	// added via commit eb8329e0a04db0061f714f033b4454326ba147f4 (v4.9-rc1~127^2~120^2~7)
	busy_time       uint64 //__u64	tcpi_busy_time;      /* Time (usec) busy sending data */ 								// added via commit efd90174167530c67a54273fd5d8369c87f9bd32 (v4.10-rc1~202^2~157^2~1)
	rwnd_limited    uint64 //__u64	tcpi_rwnd_limited;   /* Time (usec) limited by receive window */ 						// added via commit efd90174167530c67a54273fd5d8369c87f9bd32 (v4.10-rc1~202^2~157^2~1)
	sndbuf_limited  uint64 //__u64	tcpi_sndbuf_limited; /* Time (usec) limited by send buffer */ 							// added via commit efd90174167530c67a54273fd5d8369c87f9bd32 (v4.10-rc1~202^2~157^2~1)
	delivered       uint32 //__u32	tcpi_delivered; 																		// added via commit feb5f2ec646483fb66f9ad7218b1aad2a93a2a5c (v4.18-rc1~114^2~435^2)
	delivered_ce    uint32 //__u32	tcpi_delivered_ce; 																		// added via commit feb5f2ec646483fb66f9ad7218b1aad2a93a2a5c (v4.18-rc1~114^2~435^2)
	bytes_sent      uint64 //__u64	tcpi_bytes_sent;     /* RFC4898 tcpEStatsPerfHCDataOctetsOut */ 						// added via commit ba113c3aa79a7f941ac162d05a3620bdc985c58d (v4.19-rc1~140^2~171^2~3)
	bytes_retrans   uint64 //__u64	tcpi_bytes_retrans;  /* RFC4898 tcpEStatsPerfOctetsRetrans */ 							// added via commit fb31c9b9f6c85b1bad569ecedbde78d9e37cd87b (v4.19-rc1~140^2~171^2~2)
	dsack_dups      uint32 //__u32	tcpi_dsack_dups;     /* RFC4898 tcpEStatsStackDSACKDups */ 								// added via commit 7e10b6554ff2ce7f86d5d3eec3af5db8db482caa (v4.19-rc1~140^2~171^2~1)
	reord_seen      uint32 //__u32	tcpi_reord_seen;     /* reordering events seen */ 										// added via commit 7ec65372ca534217b53fd208500cf7aac223a383 (v4.19-rc1~140^2~171^2)
	rcv_ooopack     uint32 //__u32	tcpi_rcv_ooopack;    /* Out-of-order packets received */ 								// added via commit f9af2dbbfe01def62765a58af7fbc488351893c3 (v5.4-rc1~131^2~10)
	snd_wnd         uint32 //__u32	tcpi_snd_wnd;	     /* peer's advertised receive window after scaling (bytes) */ 		// added via commit 8f7baad7f03543451af27f5380fc816b008aa1f2 (v5.4-rc1~131^2~9)
} //};

// This is true from Linux 5.4.0 to 5.17.0 (at least)
// In future this may be replaced with a map to support < 5.4.0 kernel versions.
const sizeOfRawTCPInfo = 232

type NullableUint8 struct {
	Valid bool
	Value uint8
}

// TCPInfo is a gopher-style unpacked representation of RawTCPInfo.
type TCPInfo struct {
	State                  uint8
	CAState                uint8
	Retransmits            uint8
	Probes                 uint8
	Backoff                uint8
	Options                uint8
	SndWScale              uint8         // RawTCPInfo.bitfield0 & 0x0f
	RcvWScale              uint8         // RawTCPInfo.bitfield0 >> 4
	DeliveryRateAppLimited bool          // RawTCPInfo.bitfield1 & 1 == 1
	FastOpenClientFail     NullableUint8 // RawTCPInfo.bitfield1 >> 1 & 0x3
	RTO                    uint32
	ATO                    uint32
	SndMSS                 uint32
	RcvMSS                 uint32
	UnAcked                uint32
	Sacked                 uint32
	Lost                   uint32
	Retrans                uint32
	Fackets                uint32
	LastDataSent           uint32
	LastAckSent            uint32
	LastDataRecv           uint32
	LastAckRecv            uint32
	PMTU                   uint32
	RcvSSThresh            uint32
	RTT                    uint32
	RTTVar                 uint32
	SndSSThresh            uint32
	SndCWnd                uint32
	AdvMSS                 uint32
	Reordering             uint32
	RcvRTT                 uint32
	RcvSpace               uint32
	TotalRetrans           uint32
	PacingRate             uint64
	MaxPacingRate          uint64
	BytesAcked             uint64
	BytesReceived          uint64
	SegsOut                uint32
	SegsIn                 uint32
	NotsentBytes           uint32
	MinRTT                 uint32
	DataSegsIn             uint32
	DataSegsOut            uint32
	DeliveryRate           uint64
	BusyTime               uint64
	RwndLimited            uint64
	SndbufLimited          uint64
	Delivered              uint32
	DeliveredCE            uint32
	BytesSent              uint64
	BytesRetrans           uint64
	DSACKDups              uint32
	ReordSeen              uint32
	RcvOOOPack             uint32
	SndWnd                 uint32
}

// Unpack copies fields from RawTCPInfo to TCPInfo, taking care of the bitfields and marking fields not provided
// by older kernel versions as null. In the future it may deal with varying lengths of the struct returned by the
// system call (i.e., kernels older than 5.4.0).
func (packed *RawTCPInfo) Unpack() *TCPInfo {
	var unpacked TCPInfo

	unpacked.State = packed.state
	unpacked.CAState = packed.ca_state
	unpacked.Retransmits = packed.retransmits
	unpacked.Probes = packed.probes
	unpacked.Backoff = packed.backoff
	unpacked.Options = packed.options
	unpacked.SndWScale = packed.bitfield0 & 0x0f
	unpacked.RcvWScale = packed.bitfield0 >> 4
	unpacked.DeliveryRateAppLimited = packed.bitfield1&1 == 1 // added in v4.9 (before minimum supported kernel version)
	if CheckKernelVersion(5, 5, 0) {                          // added in v5.5
		unpacked.FastOpenClientFail = NullableUint8{
			Valid: true,
			Value: (packed.bitfield1 >> 1) & 0x3,
		}
	}
	unpacked.RTO = packed.rto
	unpacked.ATO = packed.ato
	unpacked.SndMSS = packed.snd_mss
	unpacked.RcvMSS = packed.rcv_mss
	unpacked.UnAcked = packed.unacked
	unpacked.Sacked = packed.sacked
	unpacked.Lost = packed.lost
	unpacked.Retrans = packed.retrans
	unpacked.Fackets = packed.fackets
	unpacked.LastDataSent = packed.last_data_sent
	unpacked.LastAckSent = packed.last_ack_sent
	unpacked.LastDataRecv = packed.last_data_recv
	unpacked.LastAckRecv = packed.last_ack_recv
	unpacked.PMTU = packed.pmtu
	unpacked.RcvSSThresh = packed.rcv_ssthresh
	unpacked.RTT = packed.rtt
	unpacked.RTTVar = packed.rttvar
	unpacked.SndSSThresh = packed.snd_ssthresh
	unpacked.SndCWnd = packed.snd_cwnd
	unpacked.AdvMSS = packed.advmss
	unpacked.Reordering = packed.reordering
	unpacked.RcvRTT = packed.rcv_rtt
	unpacked.RcvSpace = packed.rcv_space
	unpacked.TotalRetrans = packed.total_retrans
	unpacked.PacingRate = packed.pacing_rate
	unpacked.MaxPacingRate = packed.max_pacing_rate
	unpacked.BytesAcked = packed.bytes_acked
	unpacked.BytesReceived = packed.bytes_received
	unpacked.SegsOut = packed.segs_out
	unpacked.SegsIn = packed.segs_in
	unpacked.NotsentBytes = packed.notsent_bytes
	unpacked.MinRTT = packed.min_rtt
	unpacked.DataSegsIn = packed.data_segs_in
	unpacked.DataSegsOut = packed.data_segs_out
	unpacked.DeliveryRate = packed.delivery_rate
	unpacked.BusyTime = packed.busy_time
	unpacked.RwndLimited = packed.rwnd_limited
	unpacked.SndbufLimited = packed.sndbuf_limited
	unpacked.Delivered = packed.delivered
	unpacked.DeliveredCE = packed.delivered_ce
	unpacked.BytesSent = packed.bytes_sent
	unpacked.BytesRetrans = packed.bytes_retrans
	unpacked.DSACKDups = packed.dsack_dups
	unpacked.ReordSeen = packed.reord_seen
	unpacked.RcvOOOPack = packed.rcv_ooopack
	unpacked.SndWnd = packed.snd_wnd

	return &unpacked
}

// ================================================================================================================== //

// Errors from syscall package are private, so we define our own to match the errno.
var (
	EAGAIN error = syscall.EAGAIN
	EINVAL error = syscall.EINVAL
	ENOENT error = syscall.ENOENT
)

// GetTCPInfo calls getsockopt(2) on Linux to retrieve tcp_info and unpacks that into the golang-friendly TCPInfo.
func GetTCPInfo(fd int) (*TCPInfo, error) {
	var value RawTCPInfo
	length := uint32(sizeOfRawTCPInfo)

	_, _, errNo := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(fd),
		uintptr(syscall.SOL_TCP),
		uintptr(syscall.TCP_INFO),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&length)),
		0,
	)
	if errNo != 0 {
		switch errNo {
		case syscall.EAGAIN:
			return nil, EAGAIN
		case syscall.EINVAL:
			return nil, EINVAL
		case syscall.ENOENT:
			return nil, ENOENT
		}
		return nil, errNo
	}

	return value.Unpack(), nil
}

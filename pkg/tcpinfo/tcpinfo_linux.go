//go:build linux
// +build linux

package tcpinfo

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// RawInfo has identical memory layout to Linux kernel tcp_info struct (current as of kernel 5.17.0).
// bitfield0 and bitfield1 have been added to capture the 4 packed fields. Note that bitfield1 would still
// have had the same location before tcpi_delivery_rate_app_limited and tcpi_fastopen_client_fail were added
// (in v4.9.0 and v5.5.0 respectively) because of alignment rules, so they didn't increase the length or
// shift the offsets of subsequent variables.
type RawInfo struct { // struct tcp_info {          																	                                             // unless noted below, struct fields have been around since at least (1da177e4c3f41524e886b7f1b8a0c1fc7321cac2) v2.6.12-rc2^0
	state                uint8  // 1   __U8	tcpi_state;
	ca_state             uint8  // 2   __u8	tcpi_ca_state;
	retransmits          uint8  // 3   __u8	tcpi_retransmits;
	probes               uint8  // 4   __u8	tcpi_probes;
	backoff              uint8  // 5   __u8	tcpi_backoff;
	options              uint8  // 6   __u8	tcpi_options;
	bitfield0            uint8  // 7   __u8	tcpi_snd_wscale : 4, tcpi_rcv_wscale : 4;
	bitfield1            uint8  // 8   __u8	tcpi_delivery_rate_app_limited:1, tcpi_fastopen_client_fail:2; 						                                     // added via commits eb8329e0a04db0061f714f033b4454326ba147f4 (v4.9-rc1~127^2~120^2~7) and 480274787d7e3458bc5a7cfbbbe07033984ad711 (v5.5-rc1~174^2~318) respectively
	rto                  uint32 // 12  __u32 tcpi_rto;
	ato                  uint32 // 16  __u32 tcpi_ato;
	snd_mss              uint32 // 20  __u32 tcpi_snd_mss;
	rcv_mss              uint32 // 24  __u32 tcpi_rcv_mss;
	unacked              uint32 // 28  __u32 tcpi_unacked;
	sacked               uint32 // 32  __u32 tcpi_sacked;
	lost                 uint32 // 36  __u32 tcpi_lost;
	retrans              uint32 // 40  __u32 tcpi_retrans;
	fackets              uint32 // 44  __u32 tcpi_fackets;
	last_data_sent       uint32 // 48  __u32 tcpi_last_data_sent;
	last_ack_sent        uint32 // 52  __u32 tcpi_last_ack_sent;       /* Not remembered, sorry. */
	last_data_recv       uint32 // 56  __u32 tcpi_last_data_recv;
	last_ack_recv        uint32 // 60  __u32 tcpi_last_ack_recv;
	pmtu                 uint32 // 64  __u32 tcpi_pmtu;
	rcv_ssthresh         uint32 // 68  __u32 tcpi_rcv_ssthresh;
	rtt                  uint32 // 72  __u32 tcpi_rtt;
	rttvar               uint32 // 76  __u32 tcpi_rttvar;
	snd_ssthresh         uint32 // 80  __u32 tcpi_snd_ssthresh;
	snd_cwnd             uint32 // 84  __u32 tcpi_snd_cwnd;
	advmss               uint32 // 88  __u32 tcpi_advmss;
	reordering           uint32 // 92  __u32 tcpi_reordering;
	rcv_rtt              uint32 // 96  __u32 tcpi_rcv_rtt;
	rcv_space            uint32 // 100 __u32 tcpi_rcv_space;
	total_retrans        uint32 // 104 __u32 tcpi_total_retrans;
	pacing_rate          uint64 // 112 __u64 tcpi_pacing_rate; 																	                                     // added via commit 977cb0ecf82eb6d15562573c31edebf90db35163 (v3.15-rc1~113^2~349)
	max_pacing_rate      uint64 // 120 __u64 tcpi_max_pacing_rate; 																                                     // added via commit 977cb0ecf82eb6d15562573c31edebf90db35163 (v3.15-rc1~113^2~349)
	bytes_acked          uint64 // 128 __u64 tcpi_bytes_acked;         /* RFC4898 tcpEStatsAppHCThruOctetsAcked */ 					                                 // added via commit 0df48c26d8418c5c9fba63fac15b660d70ca2f1c (v4.1-rc4~26^2~34^2~22)
	bytes_received       uint64 // 136 __u64 tcpi_bytes_received;      /* RFC4898 tcpEStatsAppHCThruOctetsReceived */ 				                                 // added via commit bdd1f9edacb5f5835d1e6276571bbbe5b88ded48 (v4.1-rc4~26^2~34^2~21)
	segs_out             uint32 // 140 __u32 tcpi_segs_out;	           /* RFC4898 tcpEStatsPerfSegsOut */ 							                                 // added via commit 2efd055c53c06b7e89c167c98069bab9afce7e59 (v4.2-rc1~130^2~238)
	segs_in              uint32 // 144 __u32 tcpi_segs_in;	           /* RFC4898 tcpEStatsPerfSegsIn */ 								                             // added via commit 2efd055c53c06b7e89c167c98069bab9afce7e59 (v4.2-rc1~130^2~238)
	notsent_bytes        uint32 // 148 __u32 tcpi_notsent_bytes;       															                                     // added via commit cd9b266095f422267bddbec88f9098b48ea548fc (v4.6-rc1~91^2~262)
	min_rtt              uint32 // 152 __u32 tcpi_min_rtt; 		      															                                     // added via commit cd9b266095f422267bddbec88f9098b48ea548fc (v4.6-rc1~91^2~262)
	data_segs_in         uint32 // 156 __u32 tcpi_data_segs_in;	       /* RFC4898 tcpEStatsDataSegsIn */ 								                             // added via commit a44d6eacdaf56f74fad699af7f4925a5f5ac0e7f (v4.6-rc1~91^2~51)
	data_segs_out        uint32 // 160 __u32 tcpi_data_segs_out;       /* RFC4898 tcpEStatsDataSegsOut */ 								                             // added via commit a44d6eacdaf56f74fad699af7f4925a5f5ac0e7f (v4.6-rc1~91^2~51)
	delivery_rate        uint64 // 168 __u64 tcpi_delivery_rate;       															                                     // added via commit eb8329e0a04db0061f714f033b4454326ba147f4 (v4.9-rc1~127^2~120^2~7)
	busy_time            uint64 // 176 __u64 tcpi_busy_time;           /* Time (usec) busy sending data */ 							                                 // added via commit efd90174167530c67a54273fd5d8369c87f9bd32 (v4.10-rc1~202^2~157^2~1)
	rwnd_limited         uint64 // 184 __u64 tcpi_rwnd_limited;        /* Time (usec) limited by receive window */ 					                                 // added via commit efd90174167530c67a54273fd5d8369c87f9bd32 (v4.10-rc1~202^2~157^2~1)
	sndbuf_limited       uint64 // 192 __u64 tcpi_sndbuf_limited;      /* Time (usec) limited by send buffer */ 						                             // added via commit efd90174167530c67a54273fd5d8369c87f9bd32 (v4.10-rc1~202^2~157^2~1)
	delivered            uint32 // 196 __u32 tcpi_delivered; 	      															                                     // added via commit feb5f2ec646483fb66f9ad7218b1aad2a93a2a5c (v4.18-rc1~114^2~435^2)
	delivered_ce         uint32 // 200 __u32 tcpi_delivered_ce;       																                                 // added via commit feb5f2ec646483fb66f9ad7218b1aad2a93a2a5c (v4.18-rc1~114^2~435^2)
	bytes_sent           uint64 // 208 __u64 tcpi_bytes_sent;          /* RFC4898 tcpEStatsPerfHCDataOctetsOut */ 					                                 // added via commit ba113c3aa79a7f941ac162d05a3620bdc985c58d (v4.19-rc1~140^2~171^2~3)
	bytes_retrans        uint64 // 216 __u64 tcpi_bytes_retrans;       /* RFC4898 tcpEStatsPerfOctetsRetrans */ 						                             // added via commit fb31c9b9f6c85b1bad569ecedbde78d9e37cd87b (v4.19-rc1~140^2~171^2~2)
	dsack_dups           uint32 // 220 __u32 tcpi_dsack_dups;          /* RFC4898 tcpEStatsStackDSACKDups */ 							                             // added via commit 7e10b6554ff2ce7f86d5d3eec3af5db8db482caa (v4.19-rc1~140^2~171^2~1)
	reord_seen           uint32 // 224 __u32 tcpi_reord_seen;          /* reordering events seen */ 									                             // added via commit 7ec65372ca534217b53fd208500cf7aac223a383 (v4.19-rc1~140^2~171^2)
	rcv_ooopack          uint32 // 228 __u32 tcpi_rcv_ooopack;         /* Out-of-order packets received */ 							                                 // added via commit f9af2dbbfe01def62765a58af7fbc488351893c3 (v5.4-rc1~131^2~10)
	snd_wnd              uint32 // 232 __u32 tcpi_snd_wnd;	           /* peer's advertised receive window after scaling (bytes) */ 	                             // added via commit 8f7baad7f03543451af27f5380fc816b008aa1f2 (v5.4-rc1~131^2~9)
	rcv_wnd              uint32 // 236 __u32 tcpi_rcv_wnd;             /* local advertised receive window after scaling (bytes) */	                                 // added via commit 71fc704768f601ed3fa36310822a5e03f310f781 (v6.2-rc1~99^2~348^2)
	rehash               uint32 // 240 __u32 tcpi_rehash;              /* PLB or timeout triggered rehash attempts */					                             // added via commit 71fc704768f601ed3fa36310822a5e03f310f781 (v6.2-rc1~99^2~348^2)
	total_rto            uint16 // 242 __u16 tcpi_total_rto            /* Total number of RTO timeouts, including	SYN/SYN-ACK and recurring timeouts.	*/			 // added via commit 3868ab0f192581eff978501a05f3dc2e01541d77 (v6.7-rc1~122^2~330^2)
	total_rto_recoveries uint16 // 244 __u16 tcpi_total_rto_recoveries /* Total number of RTO recoveries, including any unfinished recovery. */                      // added via commit 3868ab0f192581eff978501a05f3dc2e01541d77 (v6.7-rc1~122^2~330^2)
	total_rto_time       uint32 // 248 __u32 tcpi_total_rto_time       /* Total time spent in RTO recoveries in milliseconds, including any unfinished recovery. */  // added via commit 3868ab0f192581eff978501a05f3dc2e01541d77 (v6.7-rc1~122^2~330^2)
} //};

type NullableBool struct {
	Valid bool
	Value bool
}

type NullableUint8 struct {
	Valid bool
	Value uint8
}

type NullableUint16 struct {
	Valid bool
	Value uint16
}

type NullableUint32 struct {
	Valid bool
	Value uint32
}

type NullableUint64 struct {
	Valid bool
	Value uint64
}

type NullableDuration struct {
	Valid bool
	Value time.Duration
}

// SysInfo is a gopher-style unpacked representation of RawTCPInfo.
type SysInfo struct {
	State                  uint8            `tcpi:"name=state,prom_type=gauge,prom_help='Connection state, see include/net/tcp_states.h.'" json:"-"`
	StateName              string           `tcpi:"name=state_name,prom_type=gauge,prom_help='Connection state name, see include/net/tcp_states.h.'" json:"state"`
	CAState                uint8            `tcpi:"name=ca_state,prom_type=gauge,prom_help='Loss recovery state machine, see include/net/tcp.h.'" json:"caState,omitempty"`
	Retransmits            uint8            `tcpi:"name=retransmits,prom_type=gauge,prom_help='Number of timeouts (RTO based retransmissions) at this sequence (reset to zero on forward progress).'" json:"retransmits,omitempty"`
	Probes                 uint8            `tcpi:"name=probes,prom_type=gauge,prom_help='Consecutive zero window probes that have gone unanswered.'" json:"probes,omitempty"`
	Backoff                uint8            `tcpi:"name=backoff,prom_type=gauge,prom_help='Exponential timeout backoff counter. Increment on RTO, reset on successful RTT measurements.'" json:"backoff,omitempty"`
	TxOptions              []Option         `tcpi:"name=options,prom_type=gauge,prom_help='Bit encoded SYN options and other negotiations: TIMESTAMPS 0x1; SACK 0x2; WSCALE 0x4; ECN 0x8 - Was negotiated; ECN_SEEN - At least one ECT seen; SYN_DATA - SYN-ACK acknowledged data in SYN sent or rcvd.'" json:"txOptions,omitempty"`
	RxOptions              []Option         `tcpi:"name=peer_options,prom_type=gauge,prom_help='Bit encoded SYN options and other negotiations: TIMESTAMPS 0x1; SACK 0x2; WSCALE 0x4; ECN 0x8 - Was negotiated; ECN_SEEN - At least one ECT seen; SYN_DATA - SYN-ACK acknowledged data in SYN sent or rcvd.'" json:"rxOptions,omitempty"`
	TxWindowScale          uint8            `tcpi:"name=snd_wscale,prom_type=gauge,prom_help='Window scaling of send-half of connection (bit shift).'" json:"txWindowScale,omitempty"`
	RxWindowScale          uint8            `tcpi:"name=rcv_wscale,prom_type=gauge,prom_help='Window scaling of receive-half of connection (bit shift).'" json:"rxWindowScale,omitempty"`
	DeliveryRateAppLimited NullableBool     `tcpi:"name=delivery_rate_app_limited,prom_type=gauge,prom_help='Flag indicating that rate measurements reflect non-network bottlenecks (1.0 = true, 0.0 = false).'" json:"deliveryRateAppLimited,omitempty"`
	FastOpenClientFail     NullableUint8    `tcpi:"name=fastopen_client_fail,prom_type=gauge,prom_help='The reason why TCP fastopen failed. 0x0: unspecified; 0x1: no cookie sent; 0x2: SYN-ACK did not ack SYN data; 0x3: SYN-ACK did not ack SYN data after timeout (-1.0 if unavailable).'" json:"fastOpenClientFail,omitempty"`
	RTO                    time.Duration    `tcpi:"name=rto,prom_type=gauge,prom_help='Retransmission Timeout. Quantized to system jiffies.'" json:"rto,omitempty"`
	ATO                    time.Duration    `tcpi:"name=ato,prom_type=gauge,prom_help='Delayed ACK Timeout. Quantized to system jiffies.'" json:"ato,omitempty"`
	TxMSS                  uint32           `tcpi:"name=snd_mss,prom_type=gauge,prom_help='Current Maximum Segment Size. Note that this can be smaller than the negotiated MSS for various reasons.'" json:"txMSS,omitempty"`
	RxMSS                  uint32           `tcpi:"name=rcv_mss,prom_type=gauge,prom_help='Maximum observed segment size from the remote host. Used to trigger delayed ACKs.'" json:"rxMSS,omitempty"`
	UnAcked                uint32           `tcpi:"name=unacked,prom_type=gauge,prom_help='Number of segments between snd.nxt and snd.una. Accounting for the Pipe algorithm.'" json:"unacked,omitempty"`
	Sacked                 uint32           `tcpi:"name=sacked,prom_type=gauge,prom_help='Scoreboard segment marked SACKED by sack blocks. Accounting for the Pipe algorithm.'" json:"sacked,omitempty"`
	Lost                   uint32           `tcpi:"name=lost,prom_type=gauge,prom_help='Scoreboard segments marked lost by loss detection heuristics. Accounting for the Pipe algorithm.'" json:"lost,omitempty"`
	Retrans                uint32           `tcpi:"name=retrans,prom_type=gauge,prom_help='Scoreboard segments marked retransmitted. Accounting for the Pipe algorithm.'" json:"retrans,omitempty"`
	Fackets                uint32           `tcpi:"name=fackets,prom_type=counter,prom_help='Some counter in Forward Acknowledgment (FACK) TCP congestion control. M-Lab says this is unused?.'" json:"fackets,omitempty"`
	LastTxAt               time.Duration    `tcpi:"name=last_data_sent,prom_type=gauge,prom_help='Time since last data segment was sent. Quantized to jiffies.'" json:"lastTxAt,omitempty"`
	LastTxAckAt            time.Duration    `tcpi:"name=last_ack_sent,prom_type=gauge,prom_help='Time since last ACK was sent. Not implemented!.'" json:"lastTxAckAt,omitempty"`
	LastRxAt               time.Duration    `tcpi:"name=last_data_recv,prom_type=gauge,prom_help='Time since last data segment was received. Quantized to jiffies.'" json:"lastRxAt,omitempty"`
	LastRxAckAt            time.Duration    `tcpi:"name=last_ack_recv,prom_type=gauge,prom_help='Time since last ACK was received. Quantized to jiffies.'" json:"lastRxAckAt,omitempty"`
	PMTU                   uint32           `tcpi:"name=pmtu,prom_type=gauge,prom_help='Maximum IP Transmission Unit for this path.'" json:"pmtu,omitempty"`
	RxSSThreshold          uint32           `tcpi:"name=rcv_ssthresh,prom_type=gauge,prom_help='Current Window Clamp. Receiver algorithm to avoid allocating excessive receive buffers.'" json:"rxSSThreshold,omitempty"`
	RTT                    time.Duration    `tcpi:"name=rtt,prom_type=gauge,prom_help='Smoothed Round Trip Time (RTT). The Linux implementation differs from the standard.'" json:"rtt,omitempty"`
	RTTVar                 time.Duration    `tcpi:"name=rttvar,prom_type=gauge,prom_help='RTT variance. The Linux implementation differs from the standard.'" json:"rttVar,omitempty"`
	TxSSThreshold          uint32           `tcpi:"name=snd_ssthresh,prom_type=gauge,prom_help='Slow Start Threshold. Value controlled by the selected congestion control algorithm.'" json:"txSSThreshold,omitempty"`
	TxCWindow              uint32           `tcpi:"name=snd_cwnd,prom_type=gauge,prom_help='Congestion Window. Value controlled by the selected congestion control algorithm.'" json:"txCWindow,omitempty"`
	AdvMSS                 uint32           `tcpi:"name=advmss,prom_type=gauge,prom_help='Advertised maximum segment size.'" json:"advMSS,omitempty"`
	Reordering             uint32           `tcpi:"name=reordering,prom_type=gauge,prom_help='Maximum observed reordering distance.'" json:"reordering,omitempty"`
	RxRTT                  time.Duration    `tcpi:"name=rcv_rtt,prom_type=gauge,prom_help='Receiver Side RTT estimate.'" json:"rxRTT,omitempty"`
	RxSpace                uint32           `tcpi:"name=rcv_space,prom_type=gauge,prom_help='Space reserved for the receive queue. Typically updated by receiver side auto-tuning.'" json:"rxSpace,omitempty"`
	TotalRetrans           uint32           `tcpi:"name=total_retrans,prom_type=gauge,prom_help='Total number of segments containing retransmitted data.'" json:"totalRetrans,omitempty"`
	PacingRate             NullableUint64   `tcpi:"name=pacing_rate,prom_type=gauge,prom_help='Current Pacing Rate, nominally updated by congestion control.'" json:"pacingRate,omitempty"`
	MaxPacingRate          NullableUint64   `tcpi:"name=max_pacing_rate,prom_type=gauge,prom_help='Settable pacing rate clamp. Set with setsockopt( ..SO_MAX_PACING_RATE.. ).'" json:"maxPacingRate,omitempty"`
	BytesAcked             NullableUint64   `tcpi:"name=bytes_acked,prom_type=gauge,prom_help='The number of data bytes for which cumulative acknowledgments have been received | RFC4898 tcpEStatsAppHCThruOctetsAcked.'" json:"bytesAcked,omitempty"`
	BytesReceived          NullableUint64   `tcpi:"name=bytes_received,prom_type=counter,prom_help='The number of data bytes for which cumulative acknowledgments have been sent | RFC4898 tcpEStatsAppHCThruOctetsReceived.'" json:"bytesReceived,omitempty"`
	SegsOut                NullableUint32   `tcpi:"name=segs_out,prom_type=gauge,prom_help='The number of segments transmitted. Includes data and pure ACKs | RFC4898 tcpEStatsPerfSegsOut.'" json:"segsOut,omitempty"`
	SegsIn                 NullableUint32   `tcpi:"name=segs_in,prom_type=gauge,prom_help='The number of segments received. Includes data and pure ACKs | RFC4898 tcpEStatsPerfSegsIn.'" json:"segsIn,omitempty"`
	NotSentBytes           NullableUint32   `tcpi:"name=notsent_bytes,prom_type=gauge,prom_help='Number of bytes queued in the send buffer that have not been sent.'" json:"notSentBytes,omitempty"`
	MinRTT                 NullableDuration `tcpi:"name=min_rtt,prom_type=gauge,prom_help='Minimum RTT. From an older, pre-BBR algorithm.'" json:"minRTT,omitempty"`
	DataSegsIn             NullableUint32   `tcpi:"name=data_segs_in,prom_type=gauge,prom_help='Input segments carrying data (len>0) | RFC4898 tcpEStatsDataSegsIn (actually tcpEStatsPerfDataSegsIn).'" json:"dataSegsIn,omitempty"`
	DataSegsOut            NullableUint32   `tcpi:"name=data_segs_out,prom_type=gauge,prom_help='Transmitted segments carrying data (len>0) | RFC4898 tcpEStatsDataSegsOut (actually tcpEStatsPerfDataSegsOut).'" json:"dataSegsOut,omitempty"`
	DeliveryRate           NullableUint64   `tcpi:"name=delivery_rate,prom_type=gauge,prom_help='Observed Maximum Delivery Rate.'" json:"deliveryRate,omitempty"`
	BusyTime               NullableUint64   `tcpi:"name=busy_time,prom_type=gauge,prom_help='Time in usecs with outstanding (unacknowledged) data. Time when snd.una not equal to snd.next.'" json:"busyTime,omitempty"`
	RxWindowLimited        NullableUint64   `tcpi:"name=rwnd_limited,prom_type=gauge,prom_help='Time in usecs spent limited by/waiting for receiver window.'" json:"rwndLimited,omitempty"`
	TxBufferLimited        NullableUint64   `tcpi:"name=sndbuf_limited,prom_type=gauge,prom_help='Time in usecs spent limited by/waiting for sender buffer space. This only includes the time when TCP transmissions are starved for data, but the application has been stopped because the buffer is full and can not be grown for some reason.'" json:"sndbufLimited,omitempty"`
	Delivered              NullableUint32   `tcpi:"name=delivered,prom_type=gauge,prom_help='Data segments delivered to the receiver including retransmits. As reported by returning ACKs, used by ECN.'" json:"delivered,omitempty"`
	DeliveredCE            NullableUint32   `tcpi:"name=delivered_ce,prom_type=gauge,prom_help='ECE marked data segments delivered to the receiver including retransmits. As reported by returning ACKs, used by ECN.'" json:"deliveredCE,omitempty"`
	BytesSent              NullableUint64   `tcpi:"name=bytes_sent,prom_type=gauge,prom_help='Payload bytes sent (excludes headers, includes retransmissions) | RFC4898 tcpEStatsPerfHCDataOctetsOut.'" json:"bytesSent,omitempty"`
	BytesRetrans           NullableUint64   `tcpi:"name=bytes_retrans,prom_type=gauge,prom_help='Bytes retransmitted. May include headers and new data carried with a retransmission (for thin flows) | RFC4898 tcpEStatsPerfOctetsRetrans.'" json:"bytesRetrans,omitempty"`
	DSACKDups              NullableUint32   `tcpi:"name=dsack_dups,prom_type=gauge,prom_help='Duplicate segments reported by DSACK | RFC4898 tcpEStatsStackDSACKDups.'" json:"dsackDups,omitempty"`
	ReordSeen              NullableUint32   `tcpi:"name=reord_seen,prom_type=counter,prom_help='Received ACKs that were out of order. Estimates reordering on the return path.'" json:"reordSeen,omitempty"`
	RxOutOfOrder           NullableUint32   `tcpi:"name=rcv_ooopack,prom_type=counter,prom_help='Out-of-order packets received.'" json:"rxOutOfOrder,omitempty"`
	TxWindow               NullableUint32   `tcpi:"name=snd_wnd,prom_type=gauge,prom_help='Peers advertised receive window after scaling (bytes).'" json:"txWindow,omitempty"`
	RxWindow               NullableUint32   `tcpi:"name=rcv_wnd,prom_type=gauge,prom_help='local advertised receive window after scaling (bytes).'" json:"rxWindow,omitempty"`
	Rehash                 NullableUint32   `tcpi:"name=rehash,prom_type=gauge,prom_help='PLB or timeout triggered rehash attempts.'" json:"rehash,omitempty"`
	TotalRTO               NullableUint16   `tcpi:"name=total_rto,prom_type=counter,prom_help='Total number of RTO timeouts, including SYN/SYN-ACK and recurring timeouts.'" json:"totalRTO,omitempty"`
	TotalRTORecoveries     NullableUint16   `tcpi:"name=total_rto_recoveries,prom_type=counter,prom_help='Total number of RTO recoveries, including any unfinished recovery.'" json:"totalRTORecoveries,omitempty"`
	TotalRTOTime           NullableUint32   `tcpi:"name=total_rto_time,prom_type=counter,prom_help='Total time spent in RTO recoveries in nanoseconds, including any unfinished recovery.'" json:"totalRTOTime,omitempty"`
}

// timeFieldMultiplier is used to convert fields representing time in microseconds to time.Duration (nanoseconds).
var timeFieldMultiplier = time.Microsecond

// Unpack copies fields from RawTCPInfo to TCPInfo, taking care of the bitfields and marking fields not provided
// by older kernel versions as null. In the future it may deal with varying lengths of the struct returned by the
// system call (i.e., kernels older than 5.4.0).
func (packed *RawInfo) Unpack() *SysInfo {
	var unpacked SysInfo

	unpacked.State = packed.state
	unpacked.StateName = tcpStateMap[packed.state]

	unpacked.CAState = packed.ca_state
	unpacked.Retransmits = packed.retransmits
	unpacked.Probes = packed.probes
	unpacked.Backoff = packed.backoff
	unpacked.TxWindowScale = packed.bitfield0 & 0x0f
	unpacked.RxWindowScale = packed.bitfield0 >> 4

	unpacked.DeliveryRateAppLimited = NullableBool{Valid: false}
	if kernelVersionIsAtLeast_4_9 {
		unpacked.DeliveryRateAppLimited.Valid = true
		unpacked.DeliveryRateAppLimited.Value = packed.bitfield1&1 == 1 // added in v4.9
	}

	unpacked.FastOpenClientFail = NullableUint8{Valid: false}
	if kernelVersionIsAtLeast_5_5 { // added in v5.5
		unpacked.FastOpenClientFail.Valid = true
		unpacked.FastOpenClientFail.Value = (packed.bitfield1 >> 1) & 0x3
	}

	unpacked.RTO = time.Duration(packed.rto) * timeFieldMultiplier
	unpacked.ATO = time.Duration(packed.ato) * timeFieldMultiplier
	unpacked.TxMSS = packed.snd_mss
	unpacked.RxMSS = packed.rcv_mss
	unpacked.UnAcked = packed.unacked
	unpacked.Sacked = packed.sacked
	unpacked.Lost = packed.lost
	unpacked.Retrans = packed.retrans
	unpacked.Fackets = packed.fackets
	unpacked.LastTxAt = time.Duration(packed.last_data_sent) * timeFieldMultiplier
	unpacked.LastTxAckAt = time.Duration(packed.last_ack_sent) * timeFieldMultiplier
	unpacked.LastRxAt = time.Duration(packed.last_data_recv) * timeFieldMultiplier
	unpacked.LastRxAckAt = time.Duration(packed.last_ack_recv) * timeFieldMultiplier
	unpacked.PMTU = packed.pmtu
	unpacked.RxSSThreshold = packed.rcv_ssthresh
	unpacked.RTT = time.Duration(packed.rtt) * timeFieldMultiplier
	unpacked.RTTVar = time.Duration(packed.rttvar) * timeFieldMultiplier
	unpacked.TxSSThreshold = packed.snd_ssthresh
	unpacked.TxCWindow = packed.snd_cwnd
	unpacked.AdvMSS = packed.advmss
	unpacked.Reordering = packed.reordering
	unpacked.RxRTT = time.Duration(packed.rcv_rtt) * timeFieldMultiplier
	unpacked.RxSpace = packed.rcv_space
	unpacked.TotalRetrans = packed.total_retrans
	unpacked.PacingRate = NullableUint64{Valid: false}
	unpacked.MaxPacingRate = NullableUint64{Valid: false}
	if kernelVersionIsAtLeast_3_15 {
		unpacked.PacingRate.Valid = true
		unpacked.PacingRate.Value = packed.pacing_rate
		unpacked.MaxPacingRate.Valid = true
		unpacked.MaxPacingRate.Value = packed.max_pacing_rate
	}

	unpacked.BytesAcked = NullableUint64{Valid: false}
	unpacked.BytesReceived = NullableUint64{Valid: false}
	if kernelVersionIsAtLeast_4_1 {
		unpacked.BytesAcked.Valid = true
		unpacked.BytesAcked.Value = packed.bytes_acked
		unpacked.BytesReceived.Valid = true
		unpacked.BytesReceived.Value = packed.bytes_received
	}

	unpacked.SegsOut = NullableUint32{Valid: false}
	unpacked.SegsIn = NullableUint32{Valid: false}
	if kernelVersionIsAtLeast_4_2 {
		unpacked.SegsOut.Valid = true
		unpacked.SegsOut.Value = packed.segs_out
		unpacked.SegsIn.Valid = true
		unpacked.SegsIn.Value = packed.segs_in
	}

	unpacked.NotSentBytes = NullableUint32{Valid: false}
	unpacked.MinRTT = NullableDuration{Valid: false}
	unpacked.DataSegsIn = NullableUint32{Valid: false}
	unpacked.DataSegsOut = NullableUint32{Valid: false}
	if kernelVersionIsAtLeast_4_6 {
		unpacked.NotSentBytes.Valid = true
		unpacked.NotSentBytes.Value = packed.notsent_bytes
		unpacked.MinRTT.Valid = true
		unpacked.MinRTT.Value = time.Duration(packed.min_rtt) * timeFieldMultiplier
		unpacked.DataSegsIn.Valid = true
		unpacked.DataSegsIn.Value = packed.data_segs_in
		unpacked.DataSegsOut.Valid = true
		unpacked.DataSegsOut.Value = packed.data_segs_out
	}

	unpacked.DeliveryRate = NullableUint64{Valid: false}
	if kernelVersionIsAtLeast_4_9 {
		unpacked.DeliveryRate.Valid = true
		unpacked.DeliveryRate.Value = packed.delivery_rate
	}

	unpacked.BusyTime = NullableUint64{Valid: false}
	unpacked.RxWindowLimited = NullableUint64{Valid: false}
	unpacked.TxBufferLimited = NullableUint64{Valid: false}
	if kernelVersionIsAtLeast_4_10 {
		unpacked.BusyTime.Valid = true
		unpacked.BusyTime.Value = packed.busy_time
		unpacked.RxWindowLimited.Valid = true
		unpacked.RxWindowLimited.Value = packed.rwnd_limited
		unpacked.TxBufferLimited.Valid = true
		unpacked.TxBufferLimited.Value = packed.sndbuf_limited
	}

	unpacked.Delivered = NullableUint32{Valid: false}
	unpacked.DeliveredCE = NullableUint32{Valid: false}
	if kernelVersionIsAtLeast_4_18 {
		unpacked.Delivered.Valid = true
		unpacked.Delivered.Value = packed.delivered
		unpacked.DeliveredCE.Valid = true
		unpacked.DeliveredCE.Value = packed.delivered_ce
	}

	unpacked.BytesSent = NullableUint64{Valid: false}
	unpacked.BytesRetrans = NullableUint64{Valid: false}
	unpacked.DSACKDups = NullableUint32{Valid: false}
	unpacked.ReordSeen = NullableUint32{Valid: false}
	if kernelVersionIsAtLeast_4_19 {
		unpacked.BytesSent.Valid = true
		unpacked.BytesSent.Value = packed.bytes_sent
		unpacked.BytesRetrans.Valid = true
		unpacked.BytesRetrans.Value = packed.bytes_retrans
		unpacked.DSACKDups.Valid = true
		unpacked.DSACKDups.Value = packed.dsack_dups
		unpacked.ReordSeen.Valid = true
		unpacked.ReordSeen.Value = packed.reord_seen
	}

	unpacked.RxOutOfOrder = NullableUint32{Valid: false}
	unpacked.TxWindow = NullableUint32{Valid: false}
	if kernelVersionIsAtLeast_5_4 {
		unpacked.RxOutOfOrder.Valid = true
		unpacked.RxOutOfOrder.Value = packed.rcv_ooopack
		unpacked.TxWindow.Valid = true
		unpacked.TxWindow.Value = packed.snd_wnd
	}

	unpacked.RxWindow = NullableUint32{Valid: false}
	unpacked.Rehash = NullableUint32{Valid: false}
	unpacked.TotalRTO = NullableUint16{Valid: false}
	unpacked.TotalRTORecoveries = NullableUint16{Valid: false}
	unpacked.TotalRTOTime = NullableUint32{Valid: false}
	if kernelVersionIsAtLeast_6_2 {
		unpacked.RxWindow.Valid = true
		unpacked.RxWindow.Value = packed.rcv_wnd
		unpacked.Rehash.Valid = true
		unpacked.Rehash.Value = packed.rehash
		unpacked.TotalRTO.Valid = true
		unpacked.TotalRTO.Value = packed.total_rto
		unpacked.TotalRTORecoveries.Valid = true
		unpacked.TotalRTORecoveries.Value = packed.total_rto_recoveries
		unpacked.TotalRTOTime.Valid = true
		unpacked.TotalRTOTime.Value = packed.total_rto_time
	}

	unpacked.TxOptions = []Option{}
	for _, flag := range tcpOptions {
		if packed.options&flag == 0 {
			continue
		}
		switch flag {
		case TCPI_OPT_TIMESTAMPS, TCPI_OPT_SACK, TCPI_OPT_ECN, TCPI_OPT_ECN_SEEN, TCPI_OPT_SYN_DATA, TCPI_OPT_USEC_TS, TCPI_OPT_TFO_CHILD:
			unpacked.TxOptions = append(unpacked.TxOptions, Option{Kind: tcpOptionsMap[flag], Value: 0})
			unpacked.RxOptions = append(unpacked.RxOptions, Option{Kind: tcpOptionsMap[flag], Value: 0})
		case TCPI_OPT_WSCALE:
			unpacked.TxOptions = append(unpacked.TxOptions, Option{Kind: tcpOptionsMap[flag], Value: uint64(packed.snd_wnd)})
			unpacked.RxOptions = append(unpacked.RxOptions, Option{Kind: tcpOptionsMap[flag], Value: uint64(packed.rcv_wnd)})
		}
	}

	return &unpacked
}

func (s *SysInfo) ToInfo() *Info {
	info := &Info{
		State:         s.StateName,
		TxOptions:     s.TxOptions,
		RxOptions:     s.RxOptions,
		TxMSS:         uint64(s.TxMSS),
		RxMSS:         uint64(s.RxMSS),
		RTT:           s.RTT,
		RTTVar:        s.RTTVar,
		RTO:           s.RTO,
		ATO:           s.ATO,
		LastTxAt:      s.LastTxAt,
		LastRxAt:      s.LastRxAt,
		LastTxAckAt:   s.LastTxAckAt,
		LastRxAckAt:   s.LastRxAckAt,
		RxWindow:      uint64(s.RxSpace),
		TxSSThreshold: uint64(s.TxSSThreshold),
		RxSSThreshold: uint64(s.RxSSThreshold),
		TxWindowSegs:  uint64(s.TxCWindow),
		Retransmits:   uint64(s.TotalRetrans),
		Sys:           s,
	}

	return info
}

// TCP state constants from linux net/tcp_states.h
const (
	TCP_ESTABLISHED = iota + 1
	TCP_SYN_SENT
	TCP_SYN_RECV
	TCP_FIN_WAIT1
	TCP_FIN_WAIT2
	TCP_TIME_WAIT
	TCP_CLOSE
	TCP_CLOSE_WAIT
	TCP_LAST_ACK
	TCP_LISTEN
	TCP_CLOSING
	TCP_NEW_SYN_RECV
)

var tcpStateMap = map[uint8]string{
	TCP_ESTABLISHED: "ESTABLISHED",
	TCP_SYN_SENT:    "SYN_SENT",
	TCP_SYN_RECV:    "SYN_RECV",
	TCP_FIN_WAIT1:   "FIN_WAIT1",
	TCP_FIN_WAIT2:   "FIN_WAIT2",
	TCP_TIME_WAIT:   "TIME_WAIT",
	TCP_CLOSE:       "CLOSE",
	TCP_CLOSE_WAIT:  "CLOSE_WAIT",
	TCP_LAST_ACK:    "LAST_ACK",
	TCP_LISTEN:      "LISTEN",
	TCP_CLOSING:     "CLOSING",
}

func tcpInfoTCPStateString(state uint8) string {
	if s, ok := tcpStateMap[state]; ok {
		return s
	}
	return fmt.Sprintf("UNKNOWN(%d)", state)
}

// TCP option flags from linux uapi/linux/tcp.h
const (
	TCPI_OPT_TIMESTAMPS = 1   /* Timestamps enabled */
	TCPI_OPT_SACK       = 2   /* SACK permitted */
	TCPI_OPT_WSCALE     = 4   /* Window scaling */
	TCPI_OPT_ECN        = 8   /* ECN was negotiated at TCP session init */
	TCPI_OPT_ECN_SEEN   = 16  /* Received at least one packet with ECT */
	TCPI_OPT_SYN_DATA   = 32  /* SYN-ACK acked data in SYN sent or rcvd */
	TCPI_OPT_USEC_TS    = 64  /* Timestamps are in usecs */
	TCPI_OPT_TFO_CHILD  = 128 /* Child from a Fast Open option on SYN */
)

var tcpOptionsMap = map[uint8]string{
	TCPI_OPT_TIMESTAMPS: "Timestamps",
	TCPI_OPT_SACK:       "SACK",
	TCPI_OPT_WSCALE:     "WindowScale",
	TCPI_OPT_ECN:        "ECN",
	TCPI_OPT_ECN_SEEN:   "ECNSeen",
	TCPI_OPT_SYN_DATA:   "SYNData",
	TCPI_OPT_USEC_TS:    "UsecTS",
	TCPI_OPT_TFO_CHILD:  "TFOChild",
}

var tcpOptions = []uint8{
	TCPI_OPT_TIMESTAMPS,
	TCPI_OPT_SACK,
	TCPI_OPT_WSCALE,
	TCPI_OPT_ECN,
	TCPI_OPT_ECN_SEEN,
	TCPI_OPT_SYN_DATA,
	TCPI_OPT_USEC_TS,
	TCPI_OPT_TFO_CHILD,
}

func tcpInfoTCPOptionsString(options uint8) string {
	var opts []string
	for _, flag := range tcpOptions {
		if options&flag != 0 {
			opts = append(opts, tcpOptionsMap[flag])
		}
	}
	return strings.Join(opts, ",")
}

// ================================================================================================================== //

// Errors from syscall package are private, so we define our own to match the errno.
var (
	EAGAIN error = syscall.EAGAIN
	EINVAL error = syscall.EINVAL
	ENOENT error = syscall.ENOENT
)

var ErrKernelTooOld = errors.New("tcp_info is not available on Linux prior to kernel 2.6.2")

// GetTCPInfo calls getsockopt(2) on Linux to retrieve tcp_info and unpacks that into the golang-friendly TCPInfo.
func GetTCPInfo(fd int) (*SysInfo, error) {
	if !kernelVersionIsAtLeast_2_6_2 {
		return nil, ErrKernelTooOld
	}

	var value RawInfo
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

func Supported() bool {
	return kernelVersionIsAtLeast_2_6_2
}

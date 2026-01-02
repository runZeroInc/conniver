//go:build darwin
// +build darwin

package tcpinfo

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// RawInfo mirrors the tcp_connection_info structure from xnu's bsd/netinet/tcp.h
type RawInfo struct {
	State               uint8  // tcpi_state: connection state
	SendWscale          uint8  // tcpi_snd_wscale: Window scale for send window
	RecvWscale          uint8  // tcpi_rcv_wscale: Window scale for receive window
	_                   uint8  // _xpad1: padding
	Options             uint32 // tcpi_options: TCP options supported
	Flags               uint32 // tcpi_flags: flags
	RTO                 uint32 // tcpi_rto: retransmit timeout in ms
	MaxSeg              uint32 // tcpi_maxseg: maximum segment size supported
	SendSSThresh        uint32 // tcpi_snd_ssthresh: slow start threshold in bytes
	SendCwnd            uint32 // tcpi_snd_cwnd: send congestion window in bytes
	SendWnd             uint32 // tcpi_snd_wnd: send widnow in bytes
	SendSBBytes         uint32 // tcpi_snd_sbbytes: bytes in send socket buffer, including in-flight data
	RecvWnd             uint32 // tcpi_rcv_wnd: receive window in bytes
	RTTCur              uint32 // tcpi_rttcur: most recent RTT in ms
	SRTT                uint32 // tcpi_srtt: average RTT in ms
	RTTVar              uint32 // tcpi_rttvar: RTT variance
	TFOFlags            uint32 // tcpi_tfo_flags: TCP Fast Open flags
	TxPackets           uint64 // tcpi_txpackets: number of packets sent
	TxBytes             uint64 // tcpi_txbytes: number of bytes sent
	TxRetransmitBytes   uint64 // tcpi_txretransmitbytes: number of retransmitted bytes
	RxPackets           uint64 // tcpi_rxpackets: number of packets received
	RxBytes             uint64 // tcpi_rxbytes: number of bytes received
	RxOutOfOrderBytes   uint64 // tcpi_rxoutoforderbytes: number of out-of-order bytes received
	TxRetransmitPackets uint64 // tcpi_txretransmitpackets: number of retransmitted packets
}

// TCPConnectionInfo structure from xnu's bsd/netinet/tcp.h:
//
//	struct tcp_connection_info {
//		u_int8_t        tcpi_state;     /* connection state */
//		u_int8_t        tcpi_snd_wscale; /* Window scale for send window */
//		u_int8_t        tcpi_rcv_wscale; /* Window scale for receive window */
//		u_int8_t        __pad1;
//		u_int32_t       tcpi_options;   /* TCP options supported */
// 			#define TCPCI_OPT_TIMESTAMPS    0x00000001 /* Timestamps enabled */
// 			#define TCPCI_OPT_SACK          0x00000002 /* SACK enabled */
//			#define TCPCI_OPT_WSCALE        0x00000004 /* Window scaling enabled */
//			#define TCPCI_OPT_ECN           0x00000008 /* ECN enabled */
//		u_int32_t       tcpi_flags;     /* flags */
//			#define TCPCI_FLAG_LOSSRECOVERY 0x00000001
//			#define TCPCI_FLAG_REORDERING_DETECTED  0x00000002
//		u_int32_t       tcpi_rto;       /* retransmit timeout in ms */
//		u_int32_t       tcpi_maxseg;    /* maximum segment size supported */
//		u_int32_t       tcpi_snd_ssthresh; /* slow start threshold in bytes */
//		u_int32_t       tcpi_snd_cwnd;  /* send congestion window in bytes */
//		u_int32_t       tcpi_snd_wnd;   /* send widnow in bytes */
//		u_int32_t       tcpi_snd_sbbytes; /* bytes in send socket buffer, including in-flight data */
//		u_int32_t       tcpi_rcv_wnd;   /* receive window in bytes*/
//		u_int32_t       tcpi_rttcur;    /* most recent RTT in ms */
//		u_int32_t       tcpi_srtt;      /* average RTT in ms */
//		u_int32_t       tcpi_rttvar;    /* RTT variance */
//		u_int32_t
//		    tcpi_tfo_cookie_req:1,             /* Cookie requested? */
//		    tcpi_tfo_cookie_rcv:1,             /* Cookie received? */
//		    tcpi_tfo_syn_loss:1,               /* Fallback to reg. TCP after SYN-loss */
//		    tcpi_tfo_syn_data_sent:1,             /* SYN+data has been sent out */
//		    tcpi_tfo_syn_data_acked:1,             /* SYN+data has been fully acknowledged */
//		    tcpi_tfo_syn_data_rcv:1,             /* Server received SYN+data with a valid cookie */
//		    tcpi_tfo_cookie_req_rcv:1,             /* Server received cookie-request */
//		    tcpi_tfo_cookie_sent:1,             /* Server announced cookie */
//		    tcpi_tfo_cookie_invalid:1,             /* Server received an invalid cookie */
//		    tcpi_tfo_cookie_wrong:1,             /* Our sent cookie was wrong */
//		    tcpi_tfo_no_cookie_rcv:1,             /* We did not receive a cookie upon our request */
//		    tcpi_tfo_heuristics_disable:1,             /* TFO-heuristics disabled it */
//		    tcpi_tfo_send_blackhole:1,             /* A sending-blackhole got detected */
//		    tcpi_tfo_recv_blackhole:1,             /* A receiver-blackhole got detected */
//		    tcpi_tfo_onebyte_proxy:1,             /* A proxy acknowledges all but one byte of the SYN */
//		    __pad2:17;
//		u_int64_t       tcpi_txpackets __attribute__((aligned(8)));
//		u_int64_t       tcpi_txbytes __attribute__((aligned(8)));
//		u_int64_t       tcpi_txretransmitbytes __attribute__((aligned(8)));
//		u_int64_t       tcpi_rxpackets __attribute__((aligned(8)));
//		u_int64_t       tcpi_rxbytes __attribute__((aligned(8)));
//		u_int64_t       tcpi_rxoutoforderbytes __attribute__((aligned(8)));
//		u_int64_t       tcpi_txretransmitpackets __attribute__((aligned(8)));
// }

// SysInfo is a gopher-style unpacked representation of RawTCPInfo.
type SysInfo struct {
	State               uint8    `tcpi:"name=state,prom_type=gauge,prom_help='Connection state, see bsd/netinet/tcp_fsm.h'" json:"-"`
	StateName           string   `tcpi:"name=state_name,prom_type=gauge,prom_help='Connection state name, see bsd/netinet/tcp_fsm.h'" json:"state"`
	SndWScale           uint8    `tcpi:"name=snd_wscale,prom_type=gauge,prom_help='Window scaling of send-half of connection.'" json:"sendWScale"`
	RcvWScale           uint8    `tcpi:"name=rcv_wscale,prom_type=gauge,prom_help='Window scaling of receive-half of connection.'" json:"recvWScale"`
	Options             []Option `tcpi:"name=options,prom_type=gauge,prom_help='TCP options supported.'" json:"options"`
	PeerOptions         []Option `tcpi:"name=peer_options,prom_type=gauge,prom_help='TCP options supported.'" json:"peerOptions"`
	Flags               string   `tcpi:"name=flags,prom_type=gauge,prom_help='TCP flags.'" json:"flags"`
	RTO                 uint64   `tcpi:"name=rto,prom_type=gauge,prom_help='Retransmit timeout in nanoseconds.'" json:"rto"`
	MaxSeg              uint32   `tcpi:"name=max_seg,prom_type=gauge,prom_help='Maximum segment size supported in bytes.'" json:"mss"`
	SendSSThresh        uint32   `tcpi:"name=send_ssthresh,prom_type=gauge,prom_help='Slow start threshold in bytes.'" json:"sendSSThreshold"`
	SendCwnd            uint32   `tcpi:"name=send_cwnd,prom_type=gauge,prom_help='Send congestion window in bytes.'" json:"sendCWindowBytes"`
	SendWnd             uint32   `tcpi:"name=send_wnd,prom_type=gauge,prom_help='Send window in bytes.'" json:"sendWnd"`
	SendSBBytes         uint32   `tcpi:"name=send_sbbytes,prom_type=gauge,prom_help='Bytes in send socket buffer, including in-flight data.'" json:"sendSBBytes"`
	RecvWnd             uint32   `tcpi:"name=recv_wnd,prom_type=gauge,prom_help='Receive window in bytes.'" json:"recvWnd"`
	RTTCur              uint64   `tcpi:"name=rtt_cur,prom_type=gauge,prom_help='Most recent RTT in nanoseconds.'" json:"rttCur"`
	SRTT                uint64   `tcpi:"name=srtt,prom_type=gauge,prom_help='Average RTT in nanoseconds.'" json:"rttSmoothed"`
	RTTVar              uint64   `tcpi:"name=rtt_var,prom_type=gauge,prom_help='RTT variance in nanoseconds.'" json:"rttVar"`
	TFOFlags            uint32   `tcpi:"name=tfo_flags,prom_type=gauge,prom_help='TCP Fast Open flags.'" json:"tfoFlags"`
	TxPackets           uint64   `tcpi:"name=tx_packets,prom_type=gauge,prom_help='Number of packets sent.'" json:"txPackets"`
	TxBytes             uint64   `tcpi:"name=tx_bytes,prom_type=gauge,prom_help='Number of bytes sent.'" json:"txBytes"`
	TxRetransmitBytes   uint64   `tcpi:"name=tx_retransmit_bytes,prom_type=gauge,prom_help='Number of retransmitted bytes.'" json:"txRetransmitBytes"`
	RxPackets           uint64   `tcpi:"name=rx_packets,prom_type=gauge,prom_help='Number of packets received.'" json:"rxPackets"`
	RxBytes             uint64   `tcpi:"name=rx_bytes,prom_type=gauge,prom_help='Number of bytes received.'" json:"rxBytes"`
	RxOutOfOrderBytes   uint64   `tcpi:"name=rx_out_of_order_bytes,prom_type=gauge,prom_help='Number of out-of-order bytes received.'" json:"rxOutOfOrderBytes"`
	TxRetransmitPackets uint64   `tcpi:"name=tx_retransmit_packets,prom_type=gauge,prom_help='Number of retransmitted packets.'" json:"txRetransmitPackets"`
}

// Unpack converts fields from RawInfo to SysInfo
func (packed *RawInfo) Unpack() *SysInfo {
	var unpacked SysInfo
	unpacked.State = packed.State
	unpacked.StateName = tcpStateMap[packed.State]
	unpacked.SndWScale = packed.SendWscale
	unpacked.RcvWScale = packed.RecvWscale
	unpacked.Flags = tcpInfoTCPFlagsString(packed.Flags)
	unpacked.RTO = uint64(packed.RTO) * 1_000_000 // Convert ms to ns
	unpacked.MaxSeg = packed.MaxSeg
	unpacked.SendSSThresh = packed.SendSSThresh
	unpacked.SendCwnd = packed.SendCwnd
	unpacked.SendWnd = packed.SendWnd
	unpacked.SendSBBytes = packed.SendSBBytes
	unpacked.RecvWnd = packed.RecvWnd
	unpacked.RTTCur = uint64(packed.RTTCur) * 1_000_000 // Convert ms to ns
	unpacked.SRTT = uint64(packed.SRTT) * 1_000_000     // Convert ms to ns
	unpacked.RTTVar = uint64(packed.RTTVar) * 1_000_000 // Convert ms to ns
	unpacked.TFOFlags = packed.TFOFlags
	unpacked.TxPackets = packed.TxPackets
	unpacked.TxBytes = packed.TxBytes
	unpacked.TxRetransmitBytes = packed.TxRetransmitBytes
	unpacked.RxPackets = packed.RxPackets
	unpacked.RxBytes = packed.RxBytes
	unpacked.RxOutOfOrderBytes = packed.RxOutOfOrderBytes
	unpacked.TxRetransmitPackets = packed.TxRetransmitPackets

	unpacked.Options = []Option{}
	for _, flag := range tcpOptions {
		if packed.Options&flag == 0 {
			continue
		}
		switch flag {
		case TCPCI_OPT_SACK, TCPCI_OPT_ECN, TCPCI_OPT_TIMESTAMPS:
			unpacked.Options = append(unpacked.Options, Option{Kind: tcpOptionsMap[flag], Value: 0})
			unpacked.PeerOptions = append(unpacked.PeerOptions, Option{Kind: tcpOptionsMap[flag], Value: 0})
		case TCPCI_OPT_WSCALE:
			unpacked.Options = append(unpacked.Options, Option{Kind: tcpOptionsMap[flag], Value: uint64(packed.SendWscale)})
			unpacked.PeerOptions = append(unpacked.PeerOptions, Option{Kind: tcpOptionsMap[flag], Value: uint64(packed.RecvWscale)})
		}
	}

	return &unpacked
}

func (s *SysInfo) ToInfo() *Info {
	info := &Info{
		State:             s.StateName,
		Options:           s.Options,
		PeerOptions:       s.PeerOptions,
		SenderMSS:         uint64(s.MaxSeg),
		ReceiverMSS:       uint64(s.MaxSeg),
		RTT:               time.Duration(s.SRTT),
		RTTVar:            time.Duration(s.RTTVar),
		RTO:               time.Duration(s.RTO),
		ReceiverWindow:    uint64(s.RecvWnd),
		SenderSSThreshold: uint64(s.SendSSThresh),
		SenderWindowBytes: uint64(s.SendCwnd),
		SenderWindowSegs:  uint64(s.SendWnd),
		Sys:               s,
	}

	info.Options = s.Options
	info.PeerOptions = s.PeerOptions

	return info
}

// TCP state constants from xnu bsd/netinet/ip_compat.h
const (
	TCPS_CLOSED       = 0 /* closed */
	TCPS_LISTEN       = 1 /* listening for connection */
	TCPS_SYN_SENT     = 2 /* active, have sent syn */
	TCPS_SYN_RECEIVED = 3 /* have send and received syn */
	/* states < TCPS_ESTABLISHED are those where connections not established */
	TCPS_ESTABLISHED = 4 /* established */
	TCPS_CLOSE_WAIT  = 5 /* rcvd fin, waiting for close */
	/* states > TCPS_CLOSE_WAIT are those where user has closed */
	TCPS_FIN_WAIT_1 = 6 /* have closed, sent fin */
	TCPS_CLOSING    = 7 /* closed xchd FIN; await FIN ACK */
	TCPS_LAST_ACK   = 8 /* had fin and close; await FIN ACK */
	/* states > TCPS_CLOSE_WAIT && < TCPS_FIN_WAIT_2 await ACK of FIN */
	TCPS_FIN_WAIT_2 = 9  /* have closed, fin is acked */
	TCPS_TIME_WAIT  = 10 /* in 2*msl quiet wait after close */
)

var tcpStateMap = map[uint8]string{
	TCPS_ESTABLISHED:  "ESTABLISHED",
	TCPS_SYN_SENT:     "SYN_SENT",
	TCPS_SYN_RECEIVED: "SYN_RECV",
	TCPS_FIN_WAIT_1:   "FIN_WAIT1",
	TCPS_FIN_WAIT_2:   "FIN_WAIT2",
	TCPS_TIME_WAIT:    "TIME_WAIT",
	TCPS_CLOSED:       "CLOSE",
	TCPS_CLOSE_WAIT:   "CLOSE_WAIT",
	TCPS_LAST_ACK:     "LAST_ACK",
	TCPS_LISTEN:       "LISTEN",
	TCPS_CLOSING:      "CLOSING",
}

func tcpInfoTCPStateString(state uint8) string {
	if s, ok := tcpStateMap[state]; ok {
		return s
	}
	return fmt.Sprintf("UNKNOWN(%d)", state)
}

const (
	SysFlagLossRecovery       = 0x1
	SysFlagReorderingDetected = 0x2
)

var tcpFlagsMap = map[uint32]string{
	SysFlagLossRecovery:       "LOSS_RECOVERY",
	SysFlagReorderingDetected: "REORDERING_DETECTED",
}

var tcpFlags = []uint32{
	SysFlagLossRecovery,
	SysFlagReorderingDetected,
}

func tcpInfoTCPFlagsString(options uint32) string {
	var opts []string
	for _, flag := range tcpFlags {
		if options&flag != 0 {
			opts = append(opts, tcpFlagsMap[flag])
		}
	}
	return strings.Join(opts, ",")
}

// TCP option flags from xnu bsd/netinet/tcp.h
const (
	TCPCI_OPT_TIMESTAMPS = 0x00000001 /* Timestamps enabled */
	TCPCI_OPT_SACK       = 0x00000002 /* SACK enabled */
	TCPCI_OPT_WSCALE     = 0x00000004 /* Window scaling enabled */
	TCPCI_OPT_ECN        = 0x00000008 /* ECN enabled */
)

var tcpOptionsMap = map[uint32]string{
	TCPCI_OPT_TIMESTAMPS: "Timestamps",
	TCPCI_OPT_SACK:       "SACK",
	TCPCI_OPT_WSCALE:     "WindowScale",
	TCPCI_OPT_ECN:        "ECN",
}

var tcpOptions = []uint32{
	TCPCI_OPT_TIMESTAMPS,
	TCPCI_OPT_SACK,
	TCPCI_OPT_WSCALE,
	TCPCI_OPT_ECN,
}

func tcpInfoTCPOptionsString(options uint32) string {
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

// GetTCPInfo calls getsockopt(2) on Linux to retrieve tcp_info and unpacks that into the golang-friendly TCPInfo.
func GetTCPInfo(fd int) (*SysInfo, error) {
	var value RawInfo
	length := uint32(unsafe.Sizeof(value))
	var errno syscall.Errno

	// This is slightly better than x/syscall/unix.GetsockoptTCPConnection because it accounts for the
	// TCP Fast Open flags bitfield.
	_, _, errno = syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(fd),
		syscall.IPPROTO_TCP,
		unix.TCP_CONNECTION_INFO,
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&length)),
		0,
	)
	if errno != 0 {
		switch errno {
		case syscall.EAGAIN:
			return nil, EAGAIN
		case syscall.EINVAL:
			return nil, EINVAL
		case syscall.ENOENT:
			return nil, ENOENT
		}
		return nil, errno
	}

	return value.Unpack(), nil
}

func Supported() bool {
	return true
}

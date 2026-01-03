//go:build windows
// +build windows

package tcpinfo

import (
	"fmt"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

const SIO_TCP_INFO = syscall.IOC_INOUT | syscall.IOC_VENDOR | 39

// RawInfoV0 mirrors the _TCP_INFO_v0 structure from the Windows SDK
// https://learn.microsoft.com/en-us/windows/win32/api/mstcpip/ns-mstcpip-tcp_info_v0
type RawInfoV0 struct {
	State             uint32
	Mss               uint32
	ConnectionTimeMs  uint64
	TimestampsEnabled bool
	RttUs             uint32
	MinRttUs          uint32
	BytesInFlight     uint32
	Cwnd              uint32
	SndWnd            uint32
	RcvWnd            uint32
	RcvBuf            uint32
	BytesOut          uint64
	BytesIn           uint64
	BytesReordered    uint32
	BytesRetrans      uint32
	FastRetrans       uint32
	DupAcksIn         uint32
	TimeoutEpisodes   uint32
	SynRetrans        uint8
}

// RawInfoV0 mirrors the _TCP_INFO_v0 structure from the Windows SDK
// https://learn.microsoft.com/en-us/windows/win32/api/mstcpip/ns-mstcpip-tcp_info_v1
type RawInfoV1 struct {
	State             uint32
	Mss               uint32
	ConnectionTimeMs  uint64
	TimestampsEnabled bool
	RttUs             uint32
	MinRttUs          uint32
	BytesInFlight     uint32
	Cwnd              uint32
	SndWnd            uint32
	RcvWnd            uint32
	RcvBuf            uint32
	BytesOut          uint64
	BytesIn           uint64
	BytesReordered    uint32
	BytesRetrans      uint32
	FastRetrans       uint32
	DupAcksIn         uint32
	TimeoutEpisodes   uint32
	SynRetrans        uint8
	// New fields in v1
	SndLimTransRwin uint32
	SndLimTimeRwin  uint32
	SndLimBytesRwin uint64
	SndLimTransCwnd uint32
	SndLimTimeCwnd  uint32
	SndLimBytesCwnd uint64
	SndLimTransSnd  uint32
	SndLimTimeSnd   uint32
	SndLimBytesSnd  uint64
}

// SysInfo is a gopher-style unpacked representation of RawTCPInfo.
type SysInfo struct {
	State             uint32        `tcpi:"name=state,prom_type=gauge,prom_help='Connection state, see bsd/netinet/tcp_fsm.h'" json:"-"`
	StateName         string        `tcpi:"name=state_name,prom_type=gauge,prom_help='Connection state name, see bsd/netinet/tcp_fsm.h'" json:"state,omitempty"`
	MSS               uint32        `tcpi:"name=mss,prom_type=gauge,prom_help='Maximum segment size supported in bytes.'" json:"mss,omitempty"`
	ConnectedTimeNS   time.Duration `tcpi:"name=connect_time_ns,prom_type=gauge,prom_help='Connection time in nanoseconds.'" json:"connectedTimeNS,omitempty"`
	RTT               time.Duration `tcpi:"name=rtt,prom_type=gauge,prom_help='Most recent RTT in nanoseconds.'" json:"rtt,omitempty"`
	RTTMin            time.Duration `tcpi:"name=rtt_min,prom_type=gauge,prom_help='Minimum RTT in nanoseconds.'" json:"rttMin,omitempty"`
	BytesInFlight     uint32        `tcpi:"name=bytes_in_flight,prom_type=gauge,prom_help='Number of bytes in flight.'" json:"bytesInFlight,omitempty"`
	CongestionWindow  uint32        `tcpi:"name=congestion_window,prom_type=gauge,prom_help='Congestion window size in bytes.'" json:"congestionWindow,omitempty"`
	TxWindow          uint32        `tcpi:"name=tx_window,prom_type=gauge,prom_help='Sender advertised window size in bytes.'" json:"txWindow,omitempty"`
	RxWindow          uint32        `tcpi:"name=rx_window,prom_type=gauge,prom_help='Receiver advertised window size in bytes.'" json:"rxWindow,omitempty"`
	RxBuffer          uint32        `tcpi:"name=rx_buffer,prom_type=gauge,prom_help='Receiver buffer size in bytes.'" json:"rxBuffer,omitempty"`
	TxBytes           uint64        `tcpi:"name=tx_bytes,prom_type=gauge,prom_help='Total number of bytes sent.'" json:"txBytes,omitempty"`
	RxBytes           uint64        `tcpi:"name=rx_bytes,prom_type=gauge,prom_help='Total number of bytes received.'" json:"rxBytes,omitempty"`
	RxOutOfOrderBytes uint32        `tcpi:"name=rx_out_of_order_bytes,prom_type=gauge,prom_help='Total number of out-of-order bytes received.'" json:"rxOutOfOrderBytes,omitempty"`
	TxRetransmitBytes uint64        `tcpi:"name=tx_retransmit_bytes,prom_type=gauge,prom_help='Total number of retransmitted bytes.'" json:"txRetransmitBytes,omitempty"`
	FastRetrans       uint32        `tcpi:"name=fast_retransmissions,prom_type=gauge,prom_help='Number of fast retransmissions.'" json:"fastRetransmissions,omitempty"`
	DupAcksIn         uint32        `tcpi:"name=duplicate_acks_in,prom_type=gauge,prom_help='Number of duplicate ACKs received.'" json:"duplicateAcksIn,omitempty"`
	TimeoutEpisodes   uint32        `tcpi:"name=timeout_episodes,prom_type=gauge,prom_help='Number of timeout episodes.'" json:"timeoutEpisodes,omitempty"`
	SynRetrans        uint8         `tcpi:"name=syn_retransmissions,prom_type=gauge,prom_help='Number of SYN retransmissions.'" json:"synRetransmissions,omitempty"`
	// Start of v1 fields
	SndLimTransRwin     uint64        `tcpi:"name=snd_lim_trans_rwin,prom_type=gauge,prom_help='Number of segments limited by receiver window.'" json:"sndLimTransRwin,omitempty"`
	SndLimTransTimeRwin time.Duration `tcpi:"name=snd_lim_trans_time_rwin,prom_type=gauge,prom_help='Number of bytes limited by receiver window.'" json:"sndLimTransTimeRwin,omitempty"`
	SndLimBytesRwin     uint64        `tcpi:"name=snd_lim_bytes_rwin,prom_type=gauge,prom_help='Number of bytes limited by sender.'" json:"sndLimBytesRwin,omitempty"`
	SndLimTransCwnd     uint64        `tcpi:"name=snd_lim_trans_cwnd,prom_type=gauge,prom_help='Number of segments limited by congestion window.'" json:"sndLimTransCwnd,omitempty"`
	SndLimTimeCwnd      time.Duration `tcpi:"name=snd_lim_time_cwnd,prom_type=gauge,prom_help='Time limited by congestion window in milliseconds.'" json:"sndLimTimeCwnd,omitempty"`
	SndLimBytesCwnd     uint64        `tcpi:"name=snd_lim_bytes_cwnd,prom_type=gauge,prom_help='Number of bytes limited by congestion window.'" json:"sndLimBytesCwnd,omitempty"`
	SndLimTransSnd      uint64        `tcpi:"name=snd_lim_trans_snd,prom_type=gauge,prom_help='Number of segments limited by congestion window.'" json:"sndLimTransSnd,omitempty"`
	SndLimTimeSnd       time.Duration `tcpi:"name=snd_lim_time_snd,prom_type=gauge,prom_help='Time limited limited by congestion window.'" json:"sndLimTimeSnd,omitempty"`
	SndLimBytesSnd      uint64        `tcpi:"name=snd_lim_bytes_snd,prom_type=gauge,prom_help='Number of bytes limited by congestion window.'" json:"sndLimBytesSnd,omitempty"`
}

// timeFieldMultiplier is used to convert fields representing time in milliseconds to time.Duration (nanoseconds).
var timeFieldMultiplier = time.Microsecond

// Unpack converts fields from _TCP_INFO_v0 to SysInfo
func (packed *RawInfoV0) Unpack() *SysInfo {
	var unpacked SysInfo
	unpacked.State = packed.State
	unpacked.StateName = tcpStateMap[packed.State]
	unpacked.MSS = packed.Mss
	unpacked.ConnectedTimeNS = time.Duration(packed.ConnectionTimeMs) * time.Millisecond
	unpacked.RTT = time.Duration(packed.RttUs) * time.Microsecond
	unpacked.RTTMin = time.Duration(packed.MinRttUs) * time.Microsecond
	unpacked.BytesInFlight = packed.BytesInFlight
	unpacked.CongestionWindow = packed.Cwnd
	unpacked.TxWindow = packed.SndWnd
	unpacked.RxWindow = packed.RcvWnd
	unpacked.RxBuffer = packed.RcvBuf
	unpacked.TxBytes = packed.BytesOut
	unpacked.RxBytes = packed.BytesIn
	unpacked.RxOutOfOrderBytes = packed.BytesReordered
	unpacked.TxRetransmitBytes = uint64(packed.BytesRetrans)
	unpacked.FastRetrans = packed.FastRetrans
	unpacked.DupAcksIn = packed.DupAcksIn
	unpacked.TimeoutEpisodes = packed.TimeoutEpisodes
	unpacked.SynRetrans = packed.SynRetrans

	return &unpacked
}

// Unpack converts fields from _TCP_INFO_v1 to SysInfo
func (packed *RawInfoV1) Unpack() *SysInfo {
	var unpacked SysInfo
	unpacked.State = packed.State
	unpacked.StateName = tcpStateMap[packed.State]
	unpacked.MSS = packed.Mss
	unpacked.ConnectedTimeNS = time.Duration(packed.ConnectionTimeMs) * timeFieldMultiplier
	unpacked.RTT = time.Duration(packed.RttUs) * time.Microsecond
	unpacked.RTTMin = time.Duration(packed.MinRttUs) * time.Microsecond
	unpacked.BytesInFlight = packed.BytesInFlight
	unpacked.CongestionWindow = packed.Cwnd
	unpacked.TxWindow = packed.SndWnd
	unpacked.RxWindow = packed.RcvWnd
	unpacked.RxBuffer = packed.RcvBuf
	unpacked.TxBytes = packed.BytesOut
	unpacked.RxBytes = packed.BytesIn
	unpacked.RxOutOfOrderBytes = packed.BytesReordered
	unpacked.TxRetransmitBytes = uint64(packed.BytesRetrans)
	unpacked.FastRetrans = packed.FastRetrans
	unpacked.DupAcksIn = packed.DupAcksIn
	unpacked.TimeoutEpisodes = packed.TimeoutEpisodes
	unpacked.SynRetrans = packed.SynRetrans
	unpacked.SndLimTransRwin = uint64(packed.SndLimTransRwin)
	unpacked.SndLimTransTimeRwin = time.Duration(packed.SndLimTimeRwin) * time.Millisecond
	unpacked.SndLimBytesRwin = packed.SndLimBytesRwin
	unpacked.SndLimTransCwnd = uint64(packed.SndLimTransCwnd)
	unpacked.SndLimTimeCwnd = time.Duration(packed.SndLimTimeCwnd) * time.Millisecond
	unpacked.SndLimBytesCwnd = packed.SndLimBytesCwnd
	unpacked.SndLimTransSnd = uint64(packed.SndLimTransSnd)
	unpacked.SndLimTimeSnd = time.Duration(packed.SndLimTimeSnd) * time.Millisecond
	unpacked.SndLimBytesSnd = packed.SndLimBytesSnd

	return &unpacked
}

func (s *SysInfo) ToInfo() *Info {
	info := &Info{
		State:        s.StateName,
		TxMSS:        uint64(s.MSS),
		RTT:          s.RTTMin,
		RxWindow:     uint64(s.RxWindow),
		TxWindowSegs: uint64(s.TxWindow),
		Retransmits:  uint64(s.SynRetrans),
		Sys:          s,
	}
	return info
}

// TCP state constants from https://learn.microsoft.com/en-us/windows/win32/api/mstcpip/ne-mstcpip-tcpstate
const (
	TCPS_CLOSED       = 0 /* closed */
	TCPS_LISTEN       = 1 /* listening for connection */
	TCPS_SYN_SENT     = 2 /* active, have sent syn */
	TCPS_SYN_RECEIVED = 3 /* have send and received syn */
	/* states < TCPS_ESTABLISHED are those where connections not established */
	TCPS_ESTABLISHED = 4 /* established */
	/* states > TCPS_CLOSE_WAIT are those where user has closed */
	TCPS_FIN_WAIT_1 = 5 /* have closed, sent fin */
	TCPS_FIN_WAIT_2 = 6 /* have closed, fin is acked */
	TCPS_CLOSE_WAIT = 7 /* rcvd fin, waiting for close */
	TCPS_CLOSING    = 8 /* closed xchd FIN; await FIN ACK */
	TCPS_LAST_ACK   = 9 /* had fin and close; await FIN ACK */
	/* states > TCPS_CLOSE_WAIT && < TCPS_FIN_WAIT_2 await ACK of FIN */
	TCPS_TIME_WAIT = 10 /* in 2*msl quiet wait after close */
)

var tcpStateMap = map[uint32]string{
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

func tcpInfoTCPStateString(state uint32) string {
	if s, ok := tcpStateMap[state]; ok {
		return s
	}
	return fmt.Sprintf("UNKNOWN(%d)", state)
}

// ================================================================================================================== //

// Errors from syscall package are private, so we define our own to match the errno.
var (
	EAGAIN error = syscall.EAGAIN
	EINVAL error = syscall.EINVAL
	ENOENT error = syscall.ENOENT
)

// GetTCPInfo calls getsockopt(2) on Linux to retrieve tcp_info and unpacks that into the golang-friendly TCPInfo.
func GetTCPInfo(fds uintptr) (*SysInfo, error) {
	fd := syscall.Handle(fds)

	// Try _TCP_INFO_v1 first
	var inbufv1 uint32 = 1
	var outbufv1 RawInfoV1

	var cbbr uint32 = 0
	var ov syscall.Overlapped

	// Try _TCP_INFO_v1 first to get extra fields
	if err := syscall.WSAIoctl(
		fd,
		SIO_TCP_INFO,
		(*byte)(unsafe.Pointer(&inbufv1)),
		uint32(unsafe.Sizeof(inbufv1)),
		(*byte)(unsafe.Pointer(&outbufv1)),
		uint32(unsafe.Sizeof(outbufv1)),
		&cbbr,
		&ov,
		0,
	); err != nil {
		// Fallback to using _TCP_INFO_v0
		var inbufv0 uint32 = 1
		var outbufv0 RawInfoV0

		if err = syscall.WSAIoctl(
			fd,
			SIO_TCP_INFO,
			(*byte)(unsafe.Pointer(&inbufv0)),
			uint32(unsafe.Sizeof(inbufv0)),
			(*byte)(unsafe.Pointer(&outbufv0)),
			uint32(unsafe.Sizeof(outbufv0)),
			&cbbr,
			&ov,
			0,
		); err != nil {
			return nil, fmt.Errorf("could not perform the WSAIoctl: %v", err)
		}
		return outbufv0.Unpack(), nil
	}

	return outbufv1.Unpack(), nil
}

func Supported() bool {
	return true
}

func (s *SysInfo) Warnings() []string {
	var warns []string
	if s.TxRetransmitBytes > 0 {
		warns = append(warns, "retransmitBytes="+strconv.FormatUint(s.TxRetransmitBytes, 10))
	}
	if s.SynRetrans > 0 {
		warns = append(warns, "retransmitSyn="+strconv.FormatUint(uint64(s.SynRetrans), 10))
	}
	if s.RxOutOfOrderBytes > 0 {
		warns = append(warns, "outOfOrderBytes="+strconv.FormatUint(uint64(s.RxOutOfOrderBytes), 10))
	}
	if s.TimeoutEpisodes > 0 {
		warns = append(warns, "timeoutEpisodes="+strconv.FormatUint(uint64(s.TimeoutEpisodes), 10))
	}
	if s.DupAcksIn > 0 {
		warns = append(warns, "duplicateAcksIn="+strconv.FormatUint(uint64(s.DupAcksIn), 10))
	}
	if s.FastRetrans > 0 {
		warns = append(warns, "fastRetransmissions="+strconv.FormatUint(uint64(s.FastRetrans), 10))
	}
	return warns
}

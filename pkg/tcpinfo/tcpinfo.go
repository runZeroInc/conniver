package tcpinfo

import (
	"fmt"
	"strconv"
	"time"
)

type Info struct {
	State         string        `json:"state,omitempty"`          // Connection state
	TxOptions     []Option      `json:"txOptions,omitempty"`      // Requesting options
	RxOptions     []Option      `json:"rxOptions,omitempty"`      // Options requested from peer
	TxMSS         uint64        `json:"txMSS,omitempty"`          // Maximum segment size for sender in bytes
	RxMSS         uint64        `json:"rxMSS,omitempty"`          // Maximum segment size for receiver in bytes
	RTT           time.Duration `json:"rtt,omitempty"`            // Round-trip time in nanoseconds
	RTTVar        time.Duration `json:"rttVar,omitempty"`         // Round-trip time variation in nanoseconds
	RTO           time.Duration `json:"rto,omitempty"`            // Retransmission timeout
	ATO           time.Duration `json:"ato,omitempty"`            // Delayed acknowledgement timeout [Linux only]
	LastTxAt      time.Duration `json:"lastTxAt,omitempty"`       // Nanoseconds since last data sent [Linux only]
	LastRxAt      time.Duration `json:"lastRxAt,omitempty"`       // Nanoseconds since last data received [FreeBSD and Linux]
	LastTxAckAt   time.Duration `json:"lastTxAckAt,omitempty"`    // Nanoseconds since last ack sent [Linux only]
	LastRxAckAt   time.Duration `json:"lastRxAckAt,omitempty"`    // Nanoseconds since last ack received [Linux only]
	RxWindow      uint64        `json:"rxWindow,omitempty"`       // Advertised receiver window in bytes
	TxSSThreshold uint64        `json:"txSSThreshold,omitempty"`  // Slow start threshold for sender in bytes or # of segments
	RxSSThreshold uint64        `json:"rxSSThreshold,omitempty"`  // Slow start threshold for receiver in bytes [Linux only]
	TxWindowBytes uint64        `json:"txCWindowBytes,omitempty"` // Congestion window for sender in bytes [Darwin and FreeBSD]
	TxWindowSegs  uint64        `json:"txCWindowSegs,omitempty"`  // Congestion window for sender in # of segments [Linux and NetBSD]
	Sys           *SysInfo      `json:"sysInfo,omitempty"`        // Platform-specific information
}

type Option struct {
	Kind  string `json:"kind"`
	Value uint64 `json:"value"`
}

func (o *Option) String() string {
	if o.Value == 0 {
		return o.Kind
	}
	return fmt.Sprintf("%s:%.2x", o.Kind, o.Value)
}

func (o *Option) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(o.String())), nil
}

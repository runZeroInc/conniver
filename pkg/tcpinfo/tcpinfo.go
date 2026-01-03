package tcpinfo

import (
	"fmt"
	"strconv"
	"time"
)

type Info struct {
	State               string        `json:"state,omitempty"`             // Connection state
	Options             []Option      `json:"options,omitempty"`           // Requesting options
	PeerOptions         []Option      `json:"peerOptions,omitempty"`       // Options requested from peer
	SenderMSS           uint64        `json:"sendMSS,omitempty"`           // Maximum segment size for sender in bytes
	ReceiverMSS         uint64        `json:"recvMSS,omitempty"`           // Maximum segment size for receiver in bytes
	RTT                 time.Duration `json:"rtt,omitempty"`               // Round-trip time in nanoseconds
	RTTVar              time.Duration `json:"rttVar,omitempty"`            // Round-trip time variation in nanoseconds
	RTO                 time.Duration `json:"rto,omitempty"`               // Retransmission timeout
	ATO                 time.Duration `json:"ato,omitempty"`               // Delayed acknowledgement timeout [Linux only]
	LastDataSent        time.Duration `json:"lastDataSent,omitempty"`      // Nanoseconds since last data sent [Linux only]
	LastDataReceived    time.Duration `json:"lastDataReceived,omitempty"`  // Nanoseconds since last data received [FreeBSD and Linux]
	LastAckReceived     time.Duration `json:"lastAckReceived,omitempty"`   // Nanoseconds since last ack received [Linux only]
	ReceiverWindow      uint64        `json:"recvWindow,omitempty"`        // Advertised receiver window in bytes
	SenderSSThreshold   uint64        `json:"sendSSThreshold,omitempty"`   // Slow start threshold for sender in bytes or # of segments
	ReceiverSSThreshold uint64        `json:"recvSSThreshold,omitempty"`   // Slow start threshold for receiver in bytes [Linux only]
	SenderWindowBytes   uint64        `json:"sendCWindowdBytes,omitempty"` // Congestion window for sender in bytes [Darwin and FreeBSD]
	SenderWindowSegs    uint64        `json:"sendCWindowSegs,omitempty"`   // Congestion window for sender in # of segments [Linux and NetBSD]
	Sys                 *SysInfo      `json:"sysInfo,omitempty"`           // Platform-specific information
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

/*
// MarshalJSON implements the MarshalJSON method of json.Marshaler
// interface.
func (i *Info) MarshalJSON() ([]byte, error) {
	raw := make(map[string]interface{})
	raw["state"] = i.State
	if len(i.Options) > 0 {
		opts := make([]string, 0, len(i.Options))
		for _, opt := range i.Options {
			opts = append(opts, opt.String())
		}
		raw["options"] = opts
	}
	if len(i.PeerOptions) > 0 {
		opts := make([]string, 0, len(i.PeerOptions))
		for _, opt := range i.PeerOptions {
			opts = append(opts, opt.String())
		}
		raw["peerOptions"] = opts
	}
	raw["sendMSS"] = i.SenderMSS
	raw["recvMSS"] = i.ReceiverMSS
	raw["rtt"] = i.RTT
	raw["rttVar"] = i.RTTVar
	raw["rto"] = i.RTO
	raw["ato"] = i.ATO
	raw["lastDataSent"] = i.LastDataSent
	raw["lastDataReceived"] = i.LastDataReceived
	raw["lastAckReceived"] = i.LastAckReceived
	raw["recvWindow"] = i.ReceiverWindow
	raw["sendSSThreshold"] = i.SenderSSThreshold
	raw["recvSSThreshold"] = i.ReceiverSSThreshold
	raw["sendWindowBytes"] = i.SenderWindowBytes
	raw["sendWindowSegs"] = i.SenderWindowSegs
	if i.Sys != nil {
		raw["sys"] = i.Sys
	}
	return json.Marshal(&raw)
}
*/

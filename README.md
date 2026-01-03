# Conniver 

Conniver is a small Go package that wraps `net.Conn` sockets and collects detailed event information.
On common platforms, the `TCP_INFO`/`TCP_CONNECTION` socket options are used to obtained kernel-level
statistics for the connection, including round-trip-time, max segment size, and more.

# Overview

Conniver is best used by specifying a DialContext with a TCP or HTTP client:

```go
import (
    "context"
    "encoding/json"
    "net/http"
    "fmt"
    
    "github.com/runZeroInc/conniver"
)
func main() {
	timeout := 15 * time.Second
	d := net.Dialer{Timeout: timeout}
	cl := &http.Client{Transport: &http.Transport{
		TLSHandshakeTimeout: timeout,
		// Set DisableKeepAlives to true to force connection close after each request.
		// Alternatively, we can call client.CloseIdleConnections() manually.
		// DisableKeepAlives:     true,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			conn, err := d.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			return conniver.WrapConn(conn, func(c *conniver.Conn, state int) {
				if state != conniver.Closed {
					return
				}
				jb, _ := json.Marshal(c)
				fmt.Println("[" + conniver.StateMap[state] + "] " + string(jb) + "\n\n")
			}), err
		},
	}}
	resp, err := cl.Get("https://www.golang.org/")
	if err != nil {
		logrus.Fatalf("get: %v", err)
	}
	_ = resp.Body.Close()

	// Use client.CloseIdleConnections() to trigger the closed events for all wrapped connections.
	// Alteratively use `DisableKeepAlives: true`` in the HTTP transport.
	cl.CloseIdleConnections()

	return
}
```

The `conniver.Conn` provides quite a few exported fields:
```go
type Conn struct {
	net.Conn                // The wrapped net.Conn
	OpenedAt        int64   // The opened time in unix nanoseconds
	ClosedAt        int64   // The closed time in unix nanoseconds
	FirstReadAt     int64   // The first successful read time in unix nanoseconds
	FirstWriteAt    int64   // The first successful write time in unix nanoseconds
	SentBytes       int64   // The number of bytes sent successfully
	RecvBytes       int64   // The number of bytes read successfully
	RecvErr         error   // The last receive error, if any 
	SentErr         error   // The last send error, if any 
	InfoErr         error   // The last tcpinfo.TCPInfo() error, if any 
	Attempts        int     // The number of retries to connect (managed by the caller)
	OpenedInfo      *tcpinfo.Info // An OS-agnostic set of TCP information fields at open time
	ClosedInfo      *tcpinfo.Info  // An OS-agnostic set of TCP information fields at close time
}
```

The `tcpinfo.Info` structure contains OS-normalized fields AND the entire platform-specific information structure.
```go
type Info struct {
	State               string        // Connection state
	Options             []Option      // Requesting options
	PeerOptions         []Option      // Options requested from peer
	SenderMSS           uint64        // Maximum segment size for sender in bytes
	ReceiverMSS         uint64        // Maximum segment size for receiver in bytes
	RTT                 time.Duration // Round-trip time in nanoseconds
	RTTVar              time.Duration // Round-trip time variation in nanoseconds
	RTO                 time.Duration // Retransmission timeout
	ATO                 time.Duration // Delayed acknowledgement timeout [Linux only]
	LastDataSent        time.Duration // Nanoseconds since last data sent [Linux only]
	LastDataReceived    time.Duration // Nanoseconds since last data received [FreeBSD and Linux]
	LastAckReceived     time.Duration // Nanoseconds since last ack received [Linux only]
	ReceiverWindow      uint64        // Advertised receiver window in bytes
	SenderSSThreshold   uint64        // Slow start threshold for sender in bytes or # of segments
	ReceiverSSThreshold uint64        // Slow start threshold for receiver in bytes [Linux only]
	SenderWindowBytes   uint64        // Congestion window for sender in bytes [Darwin and FreeBSD]
	SenderWindowSegs    uint64        // Congestion window for sender in # of segments [Linux and NetBSD]
	Sys                 *SysInfo      // Platform-specific information
}
```

The function passed to `conniver.WrapConn` is called for both the `opened` and `closed` states.
The `opened` callback fires right *after* the connection is established.
The `closed` callback fires right *before* the connection is closed.
Separate `*tcpinfo.Info{}` stats are recorded for both states.

The following reporting function will report the RTT at connection open and just before close, by
catching the `closed` event and reviewing both fields.

```go
func(c *conniver.Conn, state int) {
    if state != conniver.Closed {
        return
    }
    raw, _ := json.Marshal(c)
	fmt.Printf("Connection %s -> %s took %s, sent:%d/recv:%d bytes, starting RTT %s(%s) and ending RTT %s(%s)\n%s\n\n",
        c.LocalAddr().String(), c.RemoteAddr().String(),
        time.Duration(c.ClosedAt-c.OpenedAt),
        c.SentBytes, c.RecvBytes,
        c.OpenedInfo.RTT, c.OpenedInfo.RTTVar,
        c.ClosedInfo.RTT, c.ClosedInfo.RTTVar,
        string(raw),
    )
})
```

```json
$ go run main.go
Connection 192.168.10.23:60032 -> 216.239.36.21:443 took 273.869ms, sent:1725/recv:5897 bytes, starting RTT 6ms(3ms) and ending RTT 6ms(1ms)

{"openedAt":1767404790007006000,"closedAt":1767404790280875000,"firstReadAt":1767404790023466000,"firstWriteAt":1767404790007418000,"sentBytes":1725,"recvBytes":5897,"openedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":3000000,"recvWindow":131648,"sendSSThreshold":1073725440,"sendCWindowdBytes":14000,"sendCWindowSegs":65535,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":14000,"sendWnd":65535,"recvWnd":131648,"rttCur":6000000,"rttSmoothed":6000000,"rttVar":3000000}},"closedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":1000000,"rto":230000000,"recvWindow":125504,"sendSSThreshold":1073725440,"sendCWindowdBytes":15701,"sendCWindowSegs":267520,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"rto":230000000,"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":15701,"sendWnd":267520,"sendSBBytes":24,"recvWnd":125504,"rttCur":252000000,"rttSmoothed":6000000,"rttVar":1000000,"txPackets":5,"txBytes":1725,"rxPackets":3,"rxBytes":11497}}}

Connection 192.168.10.23:60031 -> 142.251.116.141:443 took 329.892ms, sent:1707/recv:11868 bytes, starting RTT 6ms(3ms) and ending RTT 6ms(2ms)

{"openedAt":1767404789950983000,"closedAt":1767404790280875000,"firstReadAt":1767404789958865000,"firstWriteAt":1767404789951608000,"sentBytes":1707,"recvBytes":11868,"openedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":3000000,"recvWindow":131648,"sendSSThreshold":1073725440,"sendCWindowdBytes":14000,"sendCWindowSegs":65535,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":14000,"sendWnd":65535,"recvWnd":131648,"rttCur":6000000,"rttSmoothed":6000000,"rttVar":3000000}},"closedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":2000000,"rto":230000000,"recvWindow":131072,"sendSSThreshold":1073725440,"sendCWindowdBytes":15683,"sendCWindowSegs":267520,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"rto":230000000,"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":15683,"sendWnd":267520,"sendSBBytes":24,"recvWnd":131072,"rttCur":28000000,"rttSmoothed":6000000,"rttVar":2000000,"txPackets":5,"txBytes":1707,"rxPackets":3,"rxBytes":11868}}}

```


# History

This package was bootstrapped from the following sources:

- https://github.com/simeonmiteff/go-tcpinfo/ (Mozilla Public License)
- https://github.com/mikioh/tcpinfo/ (BSD 2-Clause)
- https://github.com/mikioh/tcpopt/ (BSD 2-Clause)
- https://github.com/mikioh/tcp/ (BSD 2-Clause)

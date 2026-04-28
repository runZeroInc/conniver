# Conniver 

Conniver is a small Go package that wraps `net.Conn` sockets and collects detailed event information.
On common platforms, the `TCP_INFO`/`TCP_CONNECTION` socket options are used to obtained kernel-level
statistics for the connection, including round-trip-time, max segment size, and more.

# Overview

Conniver is best used by specifying a DialContext with a TCP or HTTP.

If you aren't able to wrap the `net.Conn`, you can instead call `tcpinfo.TCPInfo()`
directly instead. You can find the documentation for this in the package 
[README](https://github.com/runZeroInc/conniver/blob/main/pkg/tcpinfo/README.md).

The following code demonstrates using conniver to collect connection data
from the `net/http.Client`. The wrapped `net.Conn` should work with most Go
packages that allow custom dialers or provide some way to provide a
proxy `net.Conn`.

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
				// The Opened-state callback is opt-in; pass
				// conniver.WithEmitOpenCallback(true) as a third argument to
				// WrapConn if you want a notification at connect time as well.
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

# Operating Systems

The current code supports detailed TCPINFO collection for Linux, macOS, and Windows.

Support for FreeBSD is planned.

# Examples

The `conniver.Conn` struct includes basic socket details in addition to TCPInfo fields. 
```go
type Conn struct {
	net.Conn                   // The wrapped net.Conn
	Context    context.Context // The optional context
	OpenedAt   int64           // The opened time in unix nanoseconds
	ClosedAt   int64           // The closed time in unix nanoseconds
	FirstRxAt  int64           // The first successful read time in unix nanoseconds
	FirstTxAt  int64           // The first successful write time in unix nanoseconds
	LastRxAt   int64           // The last successful read time in unix nanoseconds
	LastTxAt   int64           // The last successful write time in unix nanoseconds
	TxBytes    int64           // The number of bytes sent successfully
	RxBytes    int64           // The number of bytes read successfully
	RxErr      error           // The last receive error, if any
	TxErr      error           // The last send error, if any
	InfoErr    error           // The last send error, if any
	Reconnects int             // The number of retries to connect (managed by the caller)
	OpenedInfo *tcpinfo.Info   // An OS-agnostic set of TCP information fields at open time
	ClosedInfo *tcpinfo.Info   // An OS-agnostic set of TCP information fields at close timeß
}
```

The `tcpinfo.Info` structure contains OS-normalized fields AND the entire platform-specific TCPINFO structure.
```go
type Info struct {
	State         string        // Connection state
	TxOptions     []Option      // Requesting options
	RxOptions     []Option      // Options requested from peer
	TxMSS         uint64        // Maximum segment size for sender in bytes
	RxMSS         uint64        // Maximum segment size for receiver in bytes
	RTT           time.Duration // Round-trip time in nanoseconds
	RTTVar        time.Duration // Round-trip time variation in nanoseconds
	RTO           time.Duration // Retransmission timeout
	ATO           time.Duration // Delayed acknowledgement timeout [Linux only]
	LastTxAt      time.Duration // Nanoseconds since last data sent [Linux only]
	LastRxAt      time.Duration // Nanoseconds since last data received [FreeBSD and Linux]
	LastTxAckAt   time.Duration // Nanoseconds since last ack sent [Linux only]
	LastRxAckAt   time.Duration // Nanoseconds since last ack received [Linux only]
	RxWindow      uint64        // Advertised receiver window in bytes
	TxSSThreshold uint64        // Slow start threshold for sender in bytes or # of segments
	RxSSThreshold uint64        // Slow start threshold for receiver in bytes [Linux only]
	TxWindowBytes uint64        // Congestion window for sender in bytes [Darwin and FreeBSD]
	TxWindowSegs  uint64        // Congestion window for sender in # of segments [Linux and NetBSD]
	Retransmits   uint64        // Number of retransmissions (segments or packets)
	Sys           *SysInfo      // Platform-specific information
}
```

The `*SysInfo` fields vary dramatically by operating system and require OS build tags to use directly.
The `conniver.Conn`, `tcpinfoInfo`, and `SysInfo` structs all support a `ToMap()` function, which
returns a `map[string]any` that can be used to access OS-specific fields dynamically.

The function passed to `conniver.WrapConn` is called for the `closed` state by default.
Each callback receives a detached snapshot of the wrapper state, not the live `net.Conn` wrapper itself.
The `closed` callback fires right *after* the connection has been closed, using TCP info captured
immediately before close. Open-time TCP info is collected automatically and exposed on the same
snapshot via `OpenedInfo`, so most callers only need to handle the `closed` event.

If you also want a notification at connect time, opt back into the `opened` callback by passing
`conniver.WithEmitOpenCallback(true)` to `WrapConn` / `WrapConnWithContext`. When enabled, the
`opened` callback fires right *after* the connection is established, in addition to the `closed`
callback. Separate `*tcpinfo.Info{}` stats are recorded for both states.

```go
conn = conniver.WrapConn(conn, reportFn, conniver.WithEmitOpenCallback(true))
```

The following reporting function will report the RTT at connection open and just before close, by
catching the `closed` event and reviewing both fields.

```go
func(c *conniver.Conn, state int) {
    if state != conniver.Closed {
        return
    }
    raw, _ := json.Marshal(c)
	fmt.Printf("Connection %s -> %s took %s, sent:%d/recv:%d bytes, starting RTT %s(%s) and ending RTT %s(%s)\nWarnings:%s\n%s\n\n",
		c.LocalAddrString(), c.RemoteAddrString(),
		time.Duration(c.ClosedAt-c.OpenedAt),
		c.TxBytes, c.RxBytes,
		c.OpenedInfo.RTT, c.OpenedInfo.RTTVar,
		c.ClosedInfo.RTT, c.ClosedInfo.RTTVar,
		strings.Join(c.Warnings(), ","),
        string(raw),
    )
})
```

```bash
$ go run main.go

Connection 192.168.10.23:60032 -> 216.239.36.21:443 took 273.869ms, sent:1725/recv:5897 bytes, starting RTT 6ms(3ms) and ending RTT 6ms(1ms)

{"openedAt":1767404790007006000,"closedAt":1767404790280875000,"firstReadAt":1767404790023466000,"firstWriteAt":1767404790007418000,"sentBytes":1725,"recvBytes":5897,"openedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":3000000,"recvWindow":131648,"sendSSThreshold":1073725440,"sendCWindowdBytes":14000,"sendCWindowSegs":65535,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":14000,"sendWnd":65535,"recvWnd":131648,"rttCur":6000000,"rttSmoothed":6000000,"rttVar":3000000}},"closedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":1000000,"rto":230000000,"recvWindow":125504,"sendSSThreshold":1073725440,"sendCWindowdBytes":15701,"sendCWindowSegs":267520,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"rto":230000000,"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":15701,"sendWnd":267520,"sendSBBytes":24,"recvWnd":125504,"rttCur":252000000,"rttSmoothed":6000000,"rttVar":1000000,"txPackets":5,"txBytes":1725,"rxPackets":3,"rxBytes":11497}}}

Connection 192.168.10.23:60031 -> 142.251.116.141:443 took 329.892ms, sent:1707/recv:11868 bytes, starting RTT 6ms(3ms) and ending RTT 6ms(2ms)

{"openedAt":1767404789950983000,"closedAt":1767404790280875000,"firstReadAt":1767404789958865000,"firstWriteAt":1767404789951608000,"sentBytes":1707,"recvBytes":11868,"openedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":3000000,"recvWindow":131648,"sendSSThreshold":1073725440,"sendCWindowdBytes":14000,"sendCWindowSegs":65535,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":14000,"sendWnd":65535,"recvWnd":131648,"rttCur":6000000,"rttSmoothed":6000000,"rttVar":3000000}},"closedInfo":{"state":"ESTABLISHED","options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"sendMSS":1400,"recvMSS":1400,"rtt":6000000,"rttVar":2000000,"rto":230000000,"recvWindow":131072,"sendSSThreshold":1073725440,"sendCWindowdBytes":15683,"sendCWindowSegs":267520,"sysInfo":{"state":"ESTABLISHED","sendWScale":8,"recvWScale":6,"options":["Timestamps","SACK","WindowScale:08"],"peerOptions":["Timestamps","SACK","WindowScale:06"],"rto":230000000,"mss":1400,"sendSSThreshold":1073725440,"sendCWindowBytes":15683,"sendWnd":267520,"sendSBBytes":24,"recvWnd":131072,"rttCur":28000000,"rttSmoothed":6000000,"rttVar":2000000,"txPackets":5,"txBytes":1707,"rxPackets":3,"rxBytes":11868}}}

```


# History

The `tcpinfo` package was bootstrapped from the following sources:
- https://github.com/simeonmiteff/go-tcpinfo/ (Mozilla Public License)
- https://github.com/mikioh/tcpinfo/ (BSD 2-Clause)

The `httpstat` command was derived from [httpstat](https://github.com/davecheney/httpstat) (MIT License).
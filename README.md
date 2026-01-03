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
	fmt.Printf("Connection %s -> %s took %s, sent:%d/recv:%d bytes, starting RTT %s(%s) and ending RTT %s(%s)\n",
        c.LocalAddr().String(), c.RemoteAddr().String(),
        time.Duration(c.ClosedAt-c.OpenedAt),
        c.SentBytes, c.RecvBytes,
        c.OpenedInfo.RTT, c.OpenedInfo.RTTVar,
        c.ClosedInfo.RTT, c.ClosedInfo.RTTVar,
    )
})
```

```bash
$ go run main.go
Connection 192.168.200.23:57708 -> 142.251.186.141:443 took 3.361455s, sent:1707/recv:11868 bytes, starting RTT 6ms(3ms) and ending RTT 6ms(1ms)
Connection 192.168.200.23:57709 -> 216.239.32.21:443 took 171.249ms, sent:1725/recv:7327 bytes, starting RTT 7ms(3ms) and ending RTT 6ms(2ms)
```


# History

This package was bootstrapped from the following sources:

- https://github.com/simeonmiteff/go-tcpinfo/ (Mozilla Public License)
- https://github.com/mikioh/tcpinfo/ (BSD 2-Clause)
- https://github.com/mikioh/tcpopt/ (BSD 2-Clause)
- https://github.com/mikioh/tcp/ (BSD 2-Clause)

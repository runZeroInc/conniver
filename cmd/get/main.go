package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

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
				raw, _ := json.Marshal(c)
				fmt.Printf("Connection %s -> %s took %s, sent:%d/recv:%d bytes, starting RTT %s(%s) and ending RTT %s(%s)\nWarnings:%s\n%s\n\n",
					c.LocalAddr().String(), c.RemoteAddr().String(),
					time.Duration(c.ClosedAt-c.OpenedAt),
					c.TxBytes, c.RxBytes,
					c.OpenedInfo.RTT, c.OpenedInfo.RTTVar,
					c.ClosedInfo.RTT, c.ClosedInfo.RTTVar,
					strings.Join(c.Warnings(), ", "),
					string(raw),
				)
			}), err
		},
	}}
	resp, err := cl.Get("https://www.golang.org/")
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	_ = resp.Body.Close()

	// Use client.CloseIdleConnections() to trigger the closed events for all wrapped connections.
	// Alteratively use `DisableKeepAlives: true`` in the HTTP transport.
	cl.CloseIdleConnections()
}

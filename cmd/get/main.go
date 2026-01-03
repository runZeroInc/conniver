package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/runZeroInc/conniver"
	"github.com/sirupsen/logrus"
)

const (
	SockStatsOpen  = "open"
	SockStatsClose = "close"
)

type HTTPClientWithSockStats struct {
	Client           *http.Client
	Timeout          time.Duration
	ControlContextFn func(ctx context.Context, network, address string, conn syscall.RawConn) error
	ReportFn         conniver.ReportStatsFn
}

func NewHTTPClientWithSockStats(
	timeout time.Duration,
	ctrl func(ctx context.Context, network, address string, conn syscall.RawConn) error,
	report conniver.ReportStatsFn,
) *HTTPClientWithSockStats {
	s := &HTTPClientWithSockStats{
		Timeout:          timeout,
		ControlContextFn: ctrl,
		ReportFn:         report,
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // nolint:gosec
	}

	dialer := &net.Dialer{Timeout: timeout, ControlContext: ctrl}
	transport := &http.Transport{
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: timeout,
		TLSHandshakeTimeout:   timeout,
		DisableKeepAlives:     true,
		MaxIdleConns:          0,
		TLSClientConfig:       tlsConfig,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			return s.wrapDialContext(dialer.DialContext(ctx, network, addr))
		},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	s.Client = client
	return s
}

func (s *HTTPClientWithSockStats) wrapDialContext(conn net.Conn, err error) (net.Conn, error) {
	if err != nil {
		return nil, err
	}
	return conniver.WrapConn(conn, s.ReportFn), nil
}

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
				fmt.Printf("Connection %s -> %s took %s, sent:%d/recv:%d bytes, starting RTT %s(%s) and ending RTT %s(%s)\n",
					c.LocalAddr().String(), c.RemoteAddr().String(),
					time.Duration(c.ClosedAt-c.OpenedAt),
					c.SentBytes, c.RecvBytes,
					c.OpenedInfo.RTT, c.OpenedInfo.RTTVar,
					c.ClosedInfo.RTT, c.ClosedInfo.RTTVar,
				)
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

	// TODO: Also implement HTTP clien tracing on top of the tcpinfo metrics.
	// https://pkg.go.dev/net/http/httptrace#ClientTrace
	ss := NewHTTPClientWithSockStats(15*time.Second, controlSocket, reportStats)
	t := "https://www.golang.org"

	if len(os.Args) > 1 {
		t = os.Args[1]
	}
	resp, err = ss.Client.Get(t)
	if err != nil {
		logrus.Fatalf("get: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		logrus.Fatalf("read: %v", err)
	}
	resp.Body.Close()
	time.Sleep(1 * time.Second)

	logrus.Infof("complete: %d (%s) with %d bytes", resp.StatusCode, resp.Status, len(body))
}

func controlSocket(ctx context.Context, network, address string, conn syscall.RawConn) error {
	var controlErr error
	err := conn.Control(func(fd uintptr) {
		// TBD
	})
	if err != nil {
		return err
	}
	return controlErr
}

func reportStats(tic *conniver.Conn, state int) {
	r, err := json.Marshal(tic)
	if err != nil {
		logrus.Errorf("marshal tcpinfo: %v", err)
		return
	}
	if state == conniver.Closed {
		fmt.Printf("[%s]\n%s\n\n", conniver.StateMap[state], string(r))
	}
}

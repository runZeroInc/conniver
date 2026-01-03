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

	"github.com/runZeroInc/sockstats"
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
	ReportFn         sockstats.ReportStatsFn
}

func NewHTTPClientWithSockStats(
	timeout time.Duration,
	ctrl func(ctx context.Context, network, address string, conn syscall.RawConn) error,
	report sockstats.ReportStatsFn,
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
	return sockstats.WrapConn(conn, s.ReportFn), nil
}

func main() {
	// TODO: Also implement HTTP clien tracing on top of the tcpinfo metrics.
	// https://pkg.go.dev/net/http/httptrace#ClientTrace
	ss := NewHTTPClientWithSockStats(15*time.Second, controlSocket, reportStats)
	t := "https://www.golang.org"

	if len(os.Args) > 1 {
		t = os.Args[1]
	}
	resp, err := ss.Client.Get(t)
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

func reportStats(tic *sockstats.Conn, state int) {
	r, err := json.Marshal(tic)
	if err != nil {
		logrus.Errorf("marshal tcpinfo: %v", err)
		return
	}
	if state == sockstats.Closed {
		fmt.Printf("[%s]\n%s\n\n", sockstats.StateMap[state], string(r))
	}
}

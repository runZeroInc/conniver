package conniver

import (
	"context"
	"net"
	"time"

	"github.com/runZeroInc/conniver/pkg/tcpinfo"
)

const (
	Opened = 0
	Closed = 1
)

var StateMap = map[int]string{
	Opened: "open",
	Closed: "close",
}

type ReportStatsFn func(tic *Conn, state int)

type Conn struct {
	net.Conn `json:"-"`
	Context  context.Context `json:"-"`

	reportStats     func(*Conn, int) `json:"-"`
	OpenedAt        int64            `json:"openedAt,omitempty"`
	ClosedAt        int64            `json:"closedAt,omitempty"`
	FirstReadAt     int64            `json:"firstReadAt,omitempty"`
	FirstWriteAt    int64            `json:"firstWriteAt,omitempty"`
	SentBytes       int64            `json:"sentBytes,omitempty"`
	RecvBytes       int64            `json:"recvBytes,omitempty"`
	RecvErr         error            `json:"recvErr,omitempty"`
	SentErr         error            `json:"sentErr,omitempty"`
	InfoErr         error            `json:"infoErr,omitempty"`
	Attempts        int              `json:"attempts,omitempty"`
	OpenedInfo      *tcpinfo.Info    `json:"openedInfo,omitempty"`
	ClosedInfo      *tcpinfo.Info    `json:"closedInfo,omitempty"`
	supportsTCPInfo bool
}

// WrapConn wraps the given net.Conn, triggers an immediate report in Open state,
// and returns the wrapped connection. Reads and writes are tracked and the final
// report is triggered on Close. Separate tcpinfo stats are gathered on open and
// close events.
func WrapConn(ncon net.Conn, reportStatsFn ReportStatsFn) net.Conn {
	return WrapConnWithContext(context.Background(), ncon, reportStatsFn)
}

// WrapConnWithContext wraps the given net.Conn, triggers an immediate report in Open state,
// and returns the wrapped connection. Reads and writes are tracked and the final
// report is triggered on Close. Separate tcpinfo stats are gathered on open and
// close events.
func WrapConnWithContext(ctx context.Context, ncon net.Conn, reportStatsFn ReportStatsFn) net.Conn {
	w := &Conn{
		Conn:            ncon,
		reportStats:     reportStatsFn,
		OpenedAt:        time.Now().UnixNano(),
		supportsTCPInfo: tcpinfo.Supported(),
		Context:         ctx,
	}
	w.gatherAndReport(Opened)
	return w
}

func (w *Conn) gatherAndReport(state int) {
	if w.reportStats == nil {
		return
	}

	// Only gather TCP info on open and close events once
	if state != Opened && state != Closed {
		return
	}
	if state == Opened && w.OpenedInfo != nil {
		return
	}
	if state == Closed && w.ClosedInfo != nil {
		return
	}

	// Write the report at the end regardless of success or failure
	defer w.reportStats(w, state)

	// Skipped platform or previously errored
	if !w.supportsTCPInfo || w.InfoErr != nil {
		return
	}

	tcpConn, ok := w.Conn.(*net.TCPConn)
	if !ok {
		return
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return
	}

	var sysInfo *tcpinfo.SysInfo
	if err := rawConn.Control(func(fd uintptr) {
		sysInfo, err = tcpinfo.GetTCPInfo(int(fd))
	}); err != nil {
		w.InfoErr = err
		return
	}

	if state == Opened {
		w.OpenedInfo = sysInfo.ToInfo()
		return
	}

	w.ClosedInfo = sysInfo.ToInfo()
}

// SetConnectionAttempts stores the number of attempts that were needed to open this connection.
// This is managed externally by the caller, but reported in the final stats.
func (w *Conn) SetConnectionAttempts(attempts int) {
	w.Attempts = attempts
}

// Close invokes the reportWrapper with a close event before closing the connection.
func (w *Conn) Close() error {
	w.ClosedAt = time.Now().UnixNano()
	w.gatherAndReport(Closed)
	return w.Conn.Close()
}

// Read wraps the underlying Read method and tracks the bytes received
func (w *Conn) Read(b []byte) (int, error) {
	n, err := w.Conn.Read(b)
	if err == nil && w.RecvBytes == 0 && n > 0 {
		// Track the timestamp of the first successful read
		w.FirstReadAt = time.Now().UnixNano()
	}
	w.RecvBytes += int64(n)
	if err, ok := err.(net.Error); ok && !err.Timeout() {
		w.RecvErr = err
	}
	return n, err
}

// Write wraps the underlying Write method and tracks the bytes sent
func (w *Conn) Write(b []byte) (int, error) {
	n, err := w.Conn.Write(b)
	if err == nil && w.SentBytes == 0 && n > 0 {
		// Track the timestamp of the first successful write
		w.FirstWriteAt = time.Now().UnixNano()
	}
	w.SentBytes += int64(n)
	w.SentErr = err
	if err, ok := err.(net.Error); ok && !err.Timeout() {
		w.SentErr = err
	}
	return n, err
}

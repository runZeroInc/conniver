package conniver

import (
	"context"
	"net"
	"strconv"
	"sync"
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
	FirstRxAt       int64            `json:"firstRxAt,omitempty"`
	FirstTxAt       int64            `json:"firstTxAt,omitempty"`
	LastRxAt        int64            `json:"lastRxAt,omitempty"`
	LastTxAt        int64            `json:"lastTxAt,omitempty"`
	TxBytes         int64            `json:"txBytes"`
	RxBytes         int64            `json:"rxBytes"`
	RxErr           error            `json:"rxErr,omitempty"`
	TxErr           error            `json:"txErr,omitempty"`
	InfoErr         error            `json:"infoErr,omitempty"`
	Reconnects      int              `json:"reconnects,omitempty"`
	OpenedInfo      *tcpinfo.Info    `json:"openedInfo,omitempty"`
	ClosedInfo      *tcpinfo.Info    `json:"closedInfo,omitempty"`
	supportsTCPInfo bool
	sync.Mutex
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
	var tcpErr error
	err = rawConn.Control(func(fd uintptr) {
		sysInfo, tcpErr = tcpinfo.GetTCPInfo(fd)
	})

	// Lock the struct to store the gathered info
	w.Lock()
	defer w.Unlock()

	if err != nil {
		w.InfoErr = err
		return
	}

	if tcpErr != nil {
		w.InfoErr = tcpErr
		return
	}

	if sysInfo == nil {
		return
	}

	if state == Opened {
		w.OpenedInfo = sysInfo.ToInfo()
		return
	}

	w.ClosedInfo = sysInfo.ToInfo()
}

// SetReconnects stores the number of additional connection attempts that were needed to open this connection.
// This is managed externally by the caller, but reported in the final stats.
func (w *Conn) SetReconnects(reconnects int) {
	w.Lock()
	defer w.Unlock()
	w.Reconnects = reconnects
}

// Close invokes the reportWrapper with a close event before closing the connection.
func (w *Conn) Close() error {
	w.Lock()
	w.ClosedAt = time.Now().UnixNano()
	w.Unlock()
	// The gatherAndReport function must not be called while holding the lock.
	w.gatherAndReport(Closed)
	return w.Conn.Close()
}

// Read wraps the underlying Read method and tracks the bytes received
func (w *Conn) Read(b []byte) (int, error) {
	n, err := w.Conn.Read(b)
	w.Lock()
	defer w.Unlock()
	if err == nil && n > 0 {
		ts := time.Now().UnixNano()
		if w.FirstRxAt == 0 {
			w.FirstRxAt = ts
			w.LastRxAt = ts
		} else {
			w.LastRxAt = ts
		}
	}
	w.RxBytes += int64(n)
	if err, ok := err.(net.Error); ok && !err.Timeout() {
		w.RxErr = err
	}
	return n, err
}

// Write wraps the underlying Write method and tracks the bytes sent
func (w *Conn) Write(b []byte) (int, error) {
	n, err := w.Conn.Write(b)
	w.Lock()
	defer w.Unlock()
	if err == nil && n > 0 {
		ts := time.Now().UnixNano()
		if w.FirstTxAt == 0 {
			w.FirstTxAt = ts
			w.LastTxAt = ts
		} else {
			w.LastTxAt = ts
		}
	}
	w.TxBytes += int64(n)
	w.TxErr = err
	if err, ok := err.(net.Error); ok && !err.Timeout() {
		w.TxErr = err
	}
	return n, err
}

func (w *Conn) Warnings() []string {
	w.Lock()
	defer w.Unlock()
	return w.warnings()
}

func (w *Conn) warnings() []string {
	var warns []string
	if w.Reconnects > 0 {
		warns = append(warns, "reconnects="+strconv.FormatInt(int64(w.Reconnects), 10))
	}
	for _, info := range []*tcpinfo.Info{w.OpenedInfo, w.ClosedInfo} {
		if info == nil {
			continue
		}
		if info.Retransmits > 0 {
			warns = append(warns, "retransmits="+strconv.FormatInt(int64(info.Retransmits), 10))
		}
		warns = append(warns, info.Sys.Warnings()...)
	}
	return warns
}

func (w *Conn) ToMap() map[string]any {
	w.Lock()
	defer w.Unlock()
	fset := map[string]any{
		"openedAt":   w.OpenedAt,
		"closedAt":   w.ClosedAt,
		"firstRxAt":  w.FirstRxAt,
		"firstTxAt":  w.FirstTxAt,
		"lastRxAt":   w.LastRxAt,
		"lastTxAt":   w.LastTxAt,
		"txBytes":    w.TxBytes,
		"rxBytes":    w.RxBytes,
		"reconnects": w.Reconnects,
		"localAddr":  w.LocalAddr().String(),
		"remoteAddr": w.RemoteAddr().String(),
		"warnings":   w.warnings(),
	}
	if w.RxErr != nil {
		fset["rxErr"] = w.RxErr.Error()
	}
	if w.RxErr != nil {
		fset["rxErr"] = w.RxErr.Error()
	}
	if w.TxErr != nil {
		fset["txErr"] = w.TxErr.Error()
	}
	if w.InfoErr != nil {
		fset["infoErr"] = w.InfoErr.Error()
	}
	if w.OpenedInfo != nil {
		fset["openedInfo"] = w.OpenedInfo.ToMap()
	}
	if w.ClosedInfo != nil {
		fset["closedInfo"] = w.ClosedInfo.ToMap()
	}
	return fset
}

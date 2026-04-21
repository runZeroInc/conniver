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
	closeStarted    bool
	closeDone       chan struct{}
	closeErr        error
	inFlight        int
	localAddr       net.Addr
	remoteAddr      net.Addr
	ioDrained       *sync.Cond
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
	if ncon != nil {
		w.localAddr = ncon.LocalAddr()
		w.remoteAddr = ncon.RemoteAddr()
	}
	w.ioDrained = sync.NewCond(&w.Mutex)

	openedInfo, openedInfoErr := w.collectTCPInfo()
	w.reportState(Opened, openedInfo, openedInfoErr)
	return w
}

func (w *Conn) collectTCPInfo() (*tcpinfo.Info, error) {
	if !w.supportsTCPInfo {
		return nil, nil
	}

	w.Lock()
	conn := w.Conn
	w.Unlock()

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, nil
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return nil, err
	}

	var sysInfo *tcpinfo.SysInfo
	var infoErr error
	err = rawConn.Control(func(fd uintptr) {
		sysInfo, infoErr = tcpinfo.GetTCPInfo(fd)
	})
	if err != nil {
		return nil, err
	}
	if sysInfo == nil {
		return nil, infoErr
	}
	return sysInfo.ToInfo(), infoErr
}

func (w *Conn) applyTCPInfoLocked(state int, info *tcpinfo.Info, infoErr error) {
	if info != nil {
		if state == Opened {
			w.OpenedInfo = info
		} else {
			w.ClosedInfo = info
		}
	}
	if info != nil || infoErr != nil {
		w.InfoErr = infoErr
	}
}

func (w *Conn) reportState(state int, info *tcpinfo.Info, infoErr error) {
	w.Lock()
	w.applyTCPInfoLocked(state, info, infoErr)
	reportStats := w.reportStats
	if reportStats == nil {
		w.Unlock()
		return
	}
	snapshot := w.snapshotLocked()
	if state == Opened {
		// Preserve legacy behavior for open callbacks that unwrap tic.Conn.
		snapshot.Conn = w.Conn
	}
	w.Unlock()

	reportStats(snapshot, state)
}

func (w *Conn) localAddrLocked() net.Addr {
	if w.localAddr != nil {
		return w.localAddr
	}
	if w.Conn != nil {
		return w.Conn.LocalAddr()
	}
	return nil
}

func (w *Conn) remoteAddrLocked() net.Addr {
	if w.remoteAddr != nil {
		return w.remoteAddr
	}
	if w.Conn != nil {
		return w.Conn.RemoteAddr()
	}
	return nil
}

func (w *Conn) snapshotLocked() *Conn {
	return &Conn{
		Context:         w.Context,
		OpenedAt:        w.OpenedAt,
		ClosedAt:        w.ClosedAt,
		FirstRxAt:       w.FirstRxAt,
		FirstTxAt:       w.FirstTxAt,
		LastRxAt:        w.LastRxAt,
		LastTxAt:        w.LastTxAt,
		TxBytes:         w.TxBytes,
		RxBytes:         w.RxBytes,
		RxErr:           w.RxErr,
		TxErr:           w.TxErr,
		InfoErr:         w.InfoErr,
		Reconnects:      w.Reconnects,
		OpenedInfo:      w.OpenedInfo.Clone(),
		ClosedInfo:      w.ClosedInfo.Clone(),
		supportsTCPInfo: w.supportsTCPInfo,
		closeStarted:    w.closeStarted,
		closeErr:        w.closeErr,
		localAddr:       w.localAddrLocked(),
		remoteAddr:      w.remoteAddrLocked(),
	}
}

func (w *Conn) beginIO() (net.Conn, error) {
	w.Lock()
	defer w.Unlock()

	if w.closeStarted || w.Conn == nil {
		return nil, net.ErrClosed
	}

	w.inFlight++
	return w.Conn, nil
}

func (w *Conn) finishIO() {
	w.Lock()
	defer w.Unlock()

	w.inFlight--
	if w.closeStarted && w.inFlight == 0 {
		w.ioDrained.Broadcast()
	}

	if w.inFlight < 0 {
		w.inFlight = 0
	}
}

func (w *Conn) withLiveConn(fn func(net.Conn) error) error {
	w.Lock()
	conn := w.Conn
	closeStarted := w.closeStarted
	closeErr := w.closeErr
	w.Unlock()

	if closeStarted || conn == nil {
		if closeErr != nil {
			return closeErr
		}
		return net.ErrClosed
	}

	return fn(conn)
}

// SetReconnects stores the number of additional connection attempts that were needed to open this connection.
// This is managed externally by the caller, but reported in the final stats.
func (w *Conn) SetReconnects(reconnects int) {
	w.Lock()
	defer w.Unlock()
	w.Reconnects = reconnects
}

// Close closes the underlying connection once, waits for in-flight wrapper I/O
// to finish updating stats, and invokes the callback with a detached snapshot.
func (w *Conn) Close() (err error) {
	w.Lock()
	if w.closeDone != nil {
		done := w.closeDone
		w.Unlock()
		<-done

		w.Lock()
		defer w.Unlock()
		return w.closeErr
	}
	if w.Conn == nil {
		defer w.Unlock()
		if w.closeErr != nil {
			return w.closeErr
		}
		return net.ErrClosed
	}

	w.closeStarted = true
	w.ClosedAt = time.Now().UnixNano()
	done := make(chan struct{})
	w.closeDone = done
	conn := w.Conn
	w.Unlock()

	defer close(done)

	closedInfo, closedInfoErr := w.collectTCPInfo()
	if conn != nil {
		err = conn.Close()
	} else {
		err = net.ErrClosed
	}

	w.Lock()
	w.closeErr = err
	w.Conn = nil
	for w.inFlight > 0 {
		w.ioDrained.Wait()
	}
	w.applyTCPInfoLocked(Closed, closedInfo, closedInfoErr)
	reportStats := w.reportStats
	snapshot := w.snapshotLocked()
	w.Unlock()

	if reportStats != nil {
		reportStats(snapshot, Closed)
	}

	return err
}

// Read wraps the underlying Read method and tracks the bytes received
func (w *Conn) Read(b []byte) (int, error) {
	conn, err := w.beginIO()
	if err != nil {
		return 0, err
	}

	n, err := conn.Read(b)
	w.Lock()
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
	w.Unlock()
	w.finishIO()
	return n, err
}

// Write wraps the underlying Write method and tracks the bytes sent
func (w *Conn) Write(b []byte) (int, error) {
	conn, err := w.beginIO()
	if err != nil {
		return 0, err
	}

	n, err := conn.Write(b)
	w.Lock()
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
	if err, ok := err.(net.Error); ok && !err.Timeout() {
		w.TxErr = err
	}
	w.Unlock()
	w.finishIO()
	return n, err
}

func (w *Conn) LocalAddr() net.Addr {
	w.Lock()
	defer w.Unlock()
	return w.localAddrLocked()
}

func (w *Conn) RemoteAddr() net.Addr {
	w.Lock()
	defer w.Unlock()
	return w.remoteAddrLocked()
}

func (w *Conn) LocalAddrString() string {
	w.Lock()
	defer w.Unlock()
	return addrString(w.localAddrLocked(), "unknown")
}

func (w *Conn) RemoteAddrString() string {
	w.Lock()
	defer w.Unlock()
	return addrString(w.remoteAddrLocked(), "unknown")
}

func (w *Conn) SetDeadline(t time.Time) error {
	return w.withLiveConn(func(conn net.Conn) error {
		return conn.SetDeadline(t)
	})
}

func (w *Conn) SetReadDeadline(t time.Time) error {
	return w.withLiveConn(func(conn net.Conn) error {
		return conn.SetReadDeadline(t)
	})
}

func (w *Conn) SetWriteDeadline(t time.Time) error {
	return w.withLiveConn(func(conn net.Conn) error {
		return conn.SetWriteDeadline(t)
	})
}

func (w *Conn) Warnings() []string {
	w.Lock()
	defer w.Unlock()
	return w.warnings()
}

func addrString(addr net.Addr, fallback string) string {
	if addr == nil {
		return fallback
	}
	return addr.String()
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
		if info.Sys != nil {
			warns = append(warns, info.Sys.Warnings()...)
		}
	}
	return warns
}

func (w *Conn) ToMap() map[string]any {
	w.Lock()
	defer w.Unlock()
	localAddr := w.localAddrLocked()
	remoteAddr := w.remoteAddrLocked()
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
		"localAddr":  addrString(localAddr, ""),
		"remoteAddr": addrString(remoteAddr, ""),
		"warnings":   w.warnings(),
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

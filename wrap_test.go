package conniver

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

type testAddr string

func (a testAddr) Network() string {
	return "tcp"
}

func (a testAddr) String() string {
	return string(a)
}

type fakeConn struct {
	mu sync.Mutex

	localAddr  net.Addr
	remoteAddr net.Addr

	deadline      time.Time
	readDeadline  time.Time
	writeDeadline time.Time

	closeCalls int

	closeStarted chan struct{}
	closeRelease chan struct{}
	closed       chan struct{}
	readStarted  chan struct{}
	readRelease  chan struct{}

	readData []byte
	readErr  error

	closeStartedOnce sync.Once
	closedOnce       sync.Once
	readStartedOnce  sync.Once
	readReleaseOnce  sync.Once
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		localAddr:  testAddr("127.0.0.1:12345"),
		remoteAddr: testAddr("127.0.0.1:443"),
		closed:     make(chan struct{}),
	}
}

func (c *fakeConn) Read(b []byte) (int, error) {
	if c.readStarted != nil {
		c.readStartedOnce.Do(func() {
			close(c.readStarted)
		})
	}
	if c.readRelease != nil {
		<-c.readRelease
	}
	n := copy(b, c.readData)
	return n, c.readErr
}

func (c *fakeConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func (c *fakeConn) Close() error {
	c.mu.Lock()
	c.closeCalls++
	c.mu.Unlock()

	if c.closeStarted != nil {
		c.closeStartedOnce.Do(func() {
			close(c.closeStarted)
		})
	}
	if c.readRelease != nil {
		c.readReleaseOnce.Do(func() {
			close(c.readRelease)
		})
	}
	if c.closeRelease != nil {
		<-c.closeRelease
	}
	c.closedOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *fakeConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *fakeConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *fakeConn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deadline = t
	return nil
}

func (c *fakeConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	return nil
}

func (c *fakeConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = t
	return nil
}

func (c *fakeConn) CloseCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeCalls
}

func TestConnSetDeadlinesPassThrough(t *testing.T) {
	conn := newFakeConn()
	wrapped := WrapConn(conn, nil).(*Conn)

	now := time.Now().Round(0)
	if err := wrapped.SetDeadline(now); err != nil {
		t.Fatalf("SetDeadline() error = %v", err)
	}
	if err := wrapped.SetReadDeadline(now.Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	if err := wrapped.SetWriteDeadline(now.Add(2 * time.Second)); err != nil {
		t.Fatalf("SetWriteDeadline() error = %v", err)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.deadline.Equal(now) {
		t.Fatalf("SetDeadline() = %v, want %v", conn.deadline, now)
	}
	if want := now.Add(time.Second); !conn.readDeadline.Equal(want) {
		t.Fatalf("SetReadDeadline() = %v, want %v", conn.readDeadline, want)
	}
	if want := now.Add(2 * time.Second); !conn.writeDeadline.Equal(want) {
		t.Fatalf("SetWriteDeadline() = %v, want %v", conn.writeDeadline, want)
	}
}

func TestConnAddrStringMethods(t *testing.T) {
	wrapped := WrapConn(newFakeConn(), nil).(*Conn)
	if got, want := wrapped.LocalAddrString(), "127.0.0.1:12345"; got != want {
		t.Fatalf("LocalAddrString() = %q, want %q", got, want)
	}
	if got, want := wrapped.RemoteAddrString(), "127.0.0.1:443"; got != want {
		t.Fatalf("RemoteAddrString() = %q, want %q", got, want)
	}

	empty := &Conn{}
	if got := empty.LocalAddrString(); got != "unknown" {
		t.Fatalf("LocalAddrString() on empty conn = %q, want %q", got, "unknown")
	}
	if got := empty.RemoteAddrString(); got != "unknown" {
		t.Fatalf("RemoteAddrString() on empty conn = %q, want %q", got, "unknown")
	}
}

func TestConnOpenCallbackNotFiredByDefault(t *testing.T) {
	conn := newFakeConn()
	openSnapshotCh := make(chan *Conn, 1)

	wrapped := WrapConn(conn, func(snapshot *Conn, state int) {
		if state == Opened {
			openSnapshotCh <- snapshot
		}
	}).(*Conn)

	// As of v0.0.10 the Open-state callback is opt-in. Without
	// WithEmitOpenCallback, consumers must read OpenedInfo (and other open-time
	// stats) off the snapshot delivered to the Close-state callback instead.
	select {
	case <-openSnapshotCh:
		t.Fatal("Open-state callback unexpectedly fired; it is opt-in via WithEmitOpenCallback")
	case <-time.After(50 * time.Millisecond):
	}

	// The wrapper itself is still constructed and tracking the connection.
	if got := wrapped.RemoteAddrString(); got != conn.remoteAddr.String() {
		t.Fatalf("wrapped.RemoteAddrString() = %q, want %q", got, conn.remoteAddr.String())
	}
}

func TestConnOpenCallbackFiresWhenEnabled(t *testing.T) {
	conn := newFakeConn()
	openSnapshotCh := make(chan *Conn, 1)

	WrapConn(conn, func(snapshot *Conn, state int) {
		if state == Opened {
			openSnapshotCh <- snapshot
		}
	}, WithEmitOpenCallback(true))

	select {
	case snap := <-openSnapshotCh:
		if snap == nil {
			t.Fatal("Open-state callback delivered a nil snapshot")
		}
	case <-time.After(time.Second):
		t.Fatal("Open-state callback did not fire even with WithEmitOpenCallback(true)")
	}
}

func TestConnCloseClosesUnderlyingBeforeCallbackAndOnlyOnce(t *testing.T) {
	conn := newFakeConn()
	conn.closeStarted = make(chan struct{})
	conn.closeRelease = make(chan struct{})

	callbackEntered := make(chan struct{})
	callbackRelease := make(chan struct{})

	var wrapped *Conn
	wrapped = WrapConn(conn, func(snapshot *Conn, state int) {
		if state != Closed {
			return
		}
		if snapshot == wrapped {
			t.Error("close callback received the live wrapper instead of a snapshot")
		}
		if snapshot.Conn != nil {
			t.Error("close callback snapshot should not expose the live connection")
		}
		select {
		case <-conn.closed:
		default:
			t.Error("underlying connection was not closed before the callback ran")
		}
		if got := snapshot.ToMap()["localAddr"]; got != conn.localAddr.String() {
			t.Errorf("snapshot.ToMap()[localAddr] = %v, want %v", got, conn.localAddr.String())
		}
		close(callbackEntered)
		<-callbackRelease
	}).(*Conn)

	closeErr1 := make(chan error, 1)
	closeErr2 := make(chan error, 1)

	go func() {
		closeErr1 <- wrapped.Close()
	}()

	<-conn.closeStarted

	go func() {
		closeErr2 <- wrapped.Close()
	}()

	select {
	case err := <-closeErr2:
		t.Fatalf("second Close() returned before the first close completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(conn.closeRelease)
	<-callbackEntered

	if got := conn.CloseCalls(); got != 1 {
		t.Fatalf("underlying Close() calls = %d, want 1", got)
	}

	close(callbackRelease)

	if err := <-closeErr1; err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := <-closeErr2; err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if got := conn.CloseCalls(); got != 1 {
		t.Fatalf("underlying Close() calls after both closes = %d, want 1", got)
	}
}

func TestConnCloseWaitsForInflightReadBeforeSnapshot(t *testing.T) {
	conn := newFakeConn()
	conn.readStarted = make(chan struct{})
	conn.readRelease = make(chan struct{})
	conn.readData = []byte("pong")
	conn.readErr = nil

	snapshotCh := make(chan *Conn, 1)
	wrapped := WrapConn(conn, func(snapshot *Conn, state int) {
		if state == Closed {
			snapshotCh <- snapshot
		}
	}).(*Conn)

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)

		buf := make([]byte, 8)
		n, err := wrapped.Read(buf)
		if n != len(conn.readData) {
			t.Errorf("Read() bytes = %d, want %d", n, len(conn.readData))
		}
		if err != nil {
			t.Errorf("Read() error = %v, want nil", err)
		}
		if got := string(buf[:n]); got != string(conn.readData) {
			t.Errorf("Read() data = %q, want %q", got, string(conn.readData))
		}
	}()

	<-conn.readStarted

	closeErr := make(chan error, 1)
	go func() {
		closeErr <- wrapped.Close()
	}()

	snapshot := <-snapshotCh
	<-readDone

	if err := <-closeErr; err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if snapshot.RxBytes != int64(len(conn.readData)) {
		t.Fatalf("snapshot.RxBytes = %d, want %d", snapshot.RxBytes, len(conn.readData))
	}
	if snapshot.LastRxAt == 0 {
		t.Fatal("snapshot.LastRxAt was not updated before the close callback")
	}

	buf := make([]byte, 1)
	n, err := wrapped.Read(buf)
	if n != 0 {
		t.Fatalf("Read() after Close() bytes = %d, want 0", n)
	}
	if !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Read() after Close() error = %v, want %v", err, net.ErrClosed)
	}
}

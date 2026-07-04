package conniver

import (
	"net"
	"testing"
	"time"
)

// panicConn is a net.Conn whose Read panics, modeling a caller-supplied conn
// that blows up mid-read with the panic recovered upstream.
type panicConn struct {
	net.Conn
}

func (c *panicConn) Read(b []byte) (int, error) { panic("read boom") }
func (c *panicConn) Write(b []byte) (int, error) { return len(b), nil }
func (c *panicConn) Close() error                { return nil }
func (c *panicConn) LocalAddr() net.Addr         { return testAddr("127.0.0.1:12345") }
func (c *panicConn) RemoteAddr() net.Addr        { return testAddr("127.0.0.1:443") }
func (c *panicConn) SetDeadline(time.Time) error      { return nil }
func (c *panicConn) SetReadDeadline(time.Time) error  { return nil }
func (c *panicConn) SetWriteDeadline(time.Time) error { return nil }

func TestConnCloseAfterPanickingRead(t *testing.T) {
	wrapped := WrapConn(&panicConn{}, nil)

	func() {
		defer func() { _ = recover() }()
		_, _ = wrapped.Read(make([]byte, 8))
	}()

	done := make(chan struct{})
	go func() {
		_ = wrapped.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() deadlocked after a recovered panic in Read; inFlight leaked")
	}
}

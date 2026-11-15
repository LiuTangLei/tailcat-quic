package tailcat

import (
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type proxyFailureConn struct {
	closed                          chan struct{}
	once                            sync.Once
	readErr, writeErr, halfCloseErr error
	readOne, readEOF                bool
	readOnce                        atomic.Bool
	aborts                          atomic.Int32
}

func newProxyFailureConn() *proxyFailureConn { return &proxyFailureConn{closed: make(chan struct{})} }
func (c *proxyFailureConn) Read(p []byte) (int, error) {
	if c.readErr != nil {
		return 0, c.readErr
	}
	if c.readOne && !c.readOnce.Swap(true) {
		p[0] = 42
		return 1, nil
	}
	if c.readEOF {
		return 0, io.EOF
	}
	<-c.closed
	return 0, net.ErrClosed
}
func (c *proxyFailureConn) Write(p []byte) (int, error) {
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
		return len(p), nil
	}
}
func (c *proxyFailureConn) Close() error                   { c.once.Do(func() { close(c.closed) }); return nil }
func (c *proxyFailureConn) Abort() error                   { c.aborts.Add(1); return c.Close() }
func (c *proxyFailureConn) CloseWrite() error              { return c.halfCloseErr }
func (*proxyFailureConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (*proxyFailureConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (*proxyFailureConn) SetDeadline(time.Time) error      { return nil }
func (*proxyFailureConn) SetReadDeadline(time.Time) error  { return nil }
func (*proxyFailureConn) SetWriteDeadline(time.Time) error { return nil }

func TestProxyConnsFailureClosesBoth(t *testing.T) {
	for _, mode := range []string{"read-reset", "write-failure", "half-close-failure"} {
		t.Run(mode, func(t *testing.T) {
			a, b := newProxyFailureConn(), newProxyFailureConn()
			switch mode {
			case "read-reset":
				a.readErr = errors.New("injected connection reset")
			case "write-failure":
				a.readOne = true
				b.writeErr = io.ErrClosedPipe
			case "half-close-failure":
				a.readEOF = true
				b.halfCloseErr = io.ErrClosedPipe
			}
			done := make(chan struct{})
			go func() { ProxyConns(a, b); close(done) }()
			t.Cleanup(func() {
				a.Close()
				b.Close()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("proxy workers leaked after cleanup")
				}
			})
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("one-sided failure left reverse copy blocked")
			}
			for _, c := range []*proxyFailureConn{a, b} {
				select {
				case <-c.closed:
				default:
					t.Error("failed proxy kept a connection open")
				}
				if c.aborts.Load() != 1 {
					t.Errorf("abort calls=%d; want one", c.aborts.Load())
				}
			}
		})
	}
}

type proxyPlainConn struct{ net.Conn }

func (c proxyPlainConn) CloseWrite() error { return c.Conn.(closeWriter).CloseWrite() }

func TestProxyConnsFailureClosesPlainConnections(t *testing.T) {
	a, b := newProxyFailureConn(), newProxyFailureConn()
	a.readErr = io.ErrUnexpectedEOF
	done := make(chan struct{})
	go func() { ProxyConns(proxyPlainConn{a}, proxyPlainConn{b}); close(done) }()
	t.Cleanup(func() { a.Close(); b.Close() })
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("plain net.Conn error retained reverse worker")
	}
	for _, c := range []*proxyFailureConn{a, b} {
		select {
		case <-c.closed:
		default:
			t.Error("plain connection stayed open")
		}
		if c.aborts.Load() != 0 {
			t.Fatal("plain connection unexpectedly exposed Abort")
		}
	}
}

// This measures per-connection dispatch/cleanup overhead, not network throughput.
func BenchmarkProxyConnsEOF(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		a, c := newProxyFailureConn(), newProxyFailureConn()
		a.readEOF, c.readEOF = true, true
		ProxyConns(a, c)
	}
}

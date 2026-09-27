package main

import (
	"io"
	"net"
	"runtime"
	"sync"
	"syscall/js"
	"testing"
	"time"
)

// Run the actual bridge in the Go JS/WASM runtime:
// GOOS=js GOARCH=wasm go test -exec="$(go env GOROOT)/lib/wasm/go_js_wasm_exec" web/bridge_js.go web/bridge_js_test.go
func settle(t *testing.T, promise js.Value) (js.Value, bool) {
	t.Helper()
	type result struct {
		value    js.Value
		rejected bool
	}
	ch := make(chan result, 1)
	ok := js.FuncOf(func(_ js.Value, a []js.Value) any { ch <- result{a[0], false}; return nil })
	bad := js.FuncOf(func(_ js.Value, a []js.Value) any { ch <- result{a[0], true}; return nil })
	defer ok.Release()
	defer bad.Release()
	promise.Call("then", ok, bad)
	select {
	case r := <-ch:
		return r.value, r.rejected
	case <-time.After(5 * time.Second):
		t.Fatal("promise did not settle")
		return js.Undefined(), true
	}
}

type bridgeTestConn struct {
	done                  chan struct{}
	once                  sync.Once
	blockRead, blockWrite bool
}

func (c *bridgeTestConn) Read([]byte) (int, error) {
	if c.blockRead {
		<-c.done
		return 0, io.ErrClosedPipe
	}
	return 0, io.EOF
}
func (c *bridgeTestConn) Write(b []byte) (int, error) {
	if c.blockWrite {
		<-c.done
		return 0, io.ErrClosedPipe
	}
	return len(b), nil
}
func (c *bridgeTestConn) Close() error                   { c.once.Do(func() { close(c.done) }); return nil }
func (c *bridgeTestConn) CloseWrite() error              { return nil }
func (c *bridgeTestConn) Done() <-chan struct{}          { return c.done }
func (*bridgeTestConn) LocalAddr() net.Addr              { return nil }
func (*bridgeTestConn) RemoteAddr() net.Addr             { return nil }
func (*bridgeTestConn) SetDeadline(time.Time) error      { return nil }
func (*bridgeTestConn) SetReadDeadline(time.Time) error  { return nil }
func (*bridgeTestConn) SetWriteDeadline(time.Time) error { return nil }

func ownerCount() int {
	callbackRegistry.Lock()
	defer callbackRegistry.Unlock()
	return len(callbackRegistry.owners)
}

func TestBridgeReleasesCompletedWrites(t *testing.T) {
	c := &bridgeTestConn{done: make(chan struct{})}
	closed := make(chan struct{})
	conn := makeJSConn(c, 1, func() { close(closed) })
	payload := js.Global().Get("Uint8Array").New(1 << 20)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 32 {
		if _, bad := settle(t, conn.Call("write", payload)); bad {
			t.Fatal("write rejected")
		}
	}
	conn.Call("close")
	<-closed
	runtime.GC()
	runtime.ReadMemStats(&after)
	retained := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	t.Logf("32 MiB completed; retained Go heap delta %d bytes", retained)
	if retained > 4<<20 {
		t.Fatalf("completed payloads retained: %d", retained)
	}
}

func TestBridgeCloseAndSessionShutdown(t *testing.T) {
	for _, mode := range []string{"pending-read", "pending-write", "session-shutdown"} {
		t.Run(mode, func(t *testing.T) {
			before := ownerCount()
			c := &bridgeTestConn{done: make(chan struct{}), blockRead: true, blockWrite: mode == "pending-write"}
			closed := make(chan struct{})
			conn := makeJSConn(c, 1, func() { close(closed) })
			savedClose, savedRead := conn.Get("close"), conn.Get("read")
			if _, bad := settle(t, conn.Call("closeWrite")); bad {
				t.Fatal("half-close rejected")
			}
			if ownerCount() != before+1 {
				t.Fatal("half-close released the receive side")
			}
			var pending js.Value
			if c.blockWrite {
				pending = conn.Call("write", js.Global().Get("Uint8Array").New(16))
			} else {
				pending = conn.Call("read")
			}
			if mode == "session-shutdown" {
				c.Close()
			} else {
				savedClose.Invoke()
				savedClose.Invoke()
			}
			if _, bad := settle(t, pending); !bad {
				t.Fatal("aborted I/O reported success")
			}
			select {
			case <-closed:
			case <-time.After(5 * time.Second):
				t.Fatal("close hook not called")
			}
			savedClose.Invoke()
			if _, bad := settle(t, savedRead.Invoke()); !bad {
				t.Fatal("saved method used a closed connection")
			}
			if ownerCount() != before {
				t.Fatal("connection callbacks retained")
			}
		})
	}
}

func TestBridgeRepeatedConnectionsAndListenerCallbacks(t *testing.T) {
	before := ownerCount()
	for range 100 {
		c := &bridgeTestConn{done: make(chan struct{})}
		closed := make(chan struct{})
		conn := makeJSConn(c, 1, func() { close(closed) })
		conn.Call("close")
		<-closed
		callbacks := newJSCallbacks()
		calls := 0
		closeListener := callbacks.bind("close", func([]js.Value) any { calls++; callbacks.release(); return nil })
		closeListener.Invoke()
		closeListener.Invoke()
		if calls != 1 || ownerCount() != before {
			t.Fatal("callbacks leaked or close repeated")
		}
	}
}

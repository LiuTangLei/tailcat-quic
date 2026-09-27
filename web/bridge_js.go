// Copyright (c) Tailcat-QUIC contributors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"errors"
	"io"
	"net"
	"sync"
	"syscall/js"
)

// Bind JavaScript methods to one permanent dispatcher, rather than registering
// a Go js.Func for every method of every connection. Dropping the owner removes
// all Go references; even a previously saved JS method then fails safely.
var callbackRegistry = struct {
	sync.Mutex
	next   int
	owners map[int]*jsCallbacks
}{owners: make(map[int]*jsCallbacks)}

type jsCallbacks struct {
	id      int
	methods map[string]func([]js.Value) any
}

var callbackDispatch = js.FuncOf(func(_ js.Value, args []js.Value) any {
	id, method := args[0].Int(), args[1].String()
	callbackRegistry.Lock()
	var call func([]js.Value) any
	if owner := callbackRegistry.owners[id]; owner != nil {
		call = owner.methods[method]
	}
	callbackRegistry.Unlock()
	if call == nil {
		if method == "close" {
			return nil
		}
		return rejectedPromise(io.ErrClosedPipe)
	}
	return call(args[2:])
})

func newJSCallbacks() *jsCallbacks {
	callbackRegistry.Lock()
	defer callbackRegistry.Unlock()
	callbackRegistry.next++
	owner := &jsCallbacks{id: callbackRegistry.next, methods: make(map[string]func([]js.Value) any)}
	callbackRegistry.owners[owner.id] = owner
	return owner
}

func (o *jsCallbacks) bind(name string, f func([]js.Value) any) js.Value {
	callbackRegistry.Lock()
	o.methods[name] = f
	callbackRegistry.Unlock()
	return callbackDispatch.Value.Call("bind", js.Undefined(), o.id, name)
}

func (o *jsCallbacks) release() {
	callbackRegistry.Lock()
	delete(callbackRegistry.owners, o.id)
	callbackRegistry.Unlock()
}

// makeJSConn wraps a tunneled TCP connection as a JavaScript object:
//
//	{
//	  port: number,
//	  read: () => Promise<Uint8Array|null>, // null on EOF; no concurrent calls
//	  write: (Uint8Array) => Promise,
//	  closeWrite: () => Promise, // half-close, netcat style
//	  close: () => {},
//	}
//
// read is pull-based: the browser only reads from netstack when the
// page asks for more, so a fast sender stalls on TCP backpressure
// rather than filling browser memory.
func makeJSConn(c net.Conn, port uint16, onClose func()) js.Value {
	buf := make([]byte, 64<<10)
	callbacks := newJSCallbacks()
	var closeOnce sync.Once
	closeConn := func() {
		closeOnce.Do(func() {
			callbacks.release()
			// Closing a client can await network work. Never block the JS event loop.
			go func() {
				c.Close()
				if onClose != nil {
					onClose()
				}
			}()
		})
	}
	if lifetime, ok := c.(interface{ Done() <-chan struct{} }); ok {
		go func() { <-lifetime.Done(); closeConn() }()
	}
	return js.ValueOf(map[string]any{
		"port": int(port),
		"read": callbacks.bind("read", func(args []js.Value) any {
			return makePromise(func() (any, error) {
				n, err := c.Read(buf)
				if n > 0 {
					u8 := js.Global().Get("Uint8Array").New(n)
					js.CopyBytesToJS(u8, buf[:n])
					return u8, nil
				}
				if err == nil || errors.Is(err, io.EOF) {
					return js.Null(), nil
				}
				return nil, err
			})
		}),
		"write": callbacks.bind("write", func(args []js.Value) any {
			if len(args) != 1 {
				return rejectedPromise(errors.New("write requires a Uint8Array"))
			}
			b := make([]byte, args[0].Get("length").Int())
			js.CopyBytesToGo(b, args[0])
			return makePromise(func() (any, error) {
				if _, err := c.Write(b); err != nil {
					return nil, err
				}
				return js.Undefined(), nil
			})
		}),
		"closeWrite": callbacks.bind("closeWrite", func(args []js.Value) any {
			return makePromise(func() (any, error) {
				cw, ok := c.(interface{ CloseWrite() error })
				if !ok {
					return nil, errors.New("connection does not support half-close")
				}
				if err := cw.CloseWrite(); err != nil {
					return nil, err
				}
				return js.Undefined(), nil
			})
		}),
		"close": callbacks.bind("close", func(args []js.Value) any {
			closeConn()
			return nil
		}),
	})
}

// makePromise runs f on a new goroutine and returns a JavaScript
// Promise of its result, rejected with a JavaScript Error if f
// returns an error.
func makePromise(f func() (any, error)) js.Value {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		resolve, reject := args[0], args[1]
		go func() {
			if res, err := f(); err == nil {
				resolve.Invoke(res)
			} else {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
			}
		}()
		return nil
	})
	// A Promise executes its executor synchronously, exactly once. The
	// goroutine retains its own work until settlement; the registry need not.
	defer handler.Release()
	return js.Global().Get("Promise").New(handler)
}

func rejectedPromise(err error) js.Value {
	return js.Global().Get("Promise").Call("reject", js.Global().Get("Error").New(err.Error()))
}

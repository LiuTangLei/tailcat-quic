// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package tailcat

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"tailscale.com/tstest/integration"
	"tailscale.com/types/key"
)

// TestDisconnectClientResetsOpenStream checks the H3 fork's documented
// DisconnectClient behavior: unlike upstream, where the client's WireGuard
// connections stall, an open H3 TCP stream is torn down promptly.
func TestDisconnectClientResetsOpenStream(t *testing.T) {
	t.Parallel()

	dm := integration.RunDERPAndSTUN(t, mkLogger(t, "derpstun"), "127.0.0.1")
	held := make(chan struct{})
	s := &Server{Logf: mkLogger(t, "server"), Region: dm.Regions[1]}
	s.OnTCP = func(port uint16) func(net.Conn) {
		return func(c net.Conn) {
			io.WriteString(c, "hello\n")
			<-held // keep the stream open until the test ends
			c.Close()
		}
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(held); s.Close() })

	c := &Client{Server: s.TailcatAddr(), Logf: mkLogger(t, "client")}
	t.Cleanup(func() { c.Close() })
	PingForTest(t, s, c)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := c.DialTCPPort(ctx, 80)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	buf := make([]byte, 6)
	if _, err := io.ReadFull(conn, buf); err != nil || string(buf) != "hello\n" {
		t.Fatalf("read %q, %v; want hello", buf, err)
	}

	if !s.DisconnectClient(c.PublicKey()) {
		t.Fatal("DisconnectClient reported the client as not connected")
	}
	readErr := make(chan error, 1)
	go func() {
		_, err := conn.Read(make([]byte, 1))
		readErr <- err
	}()
	select {
	case err := <-readErr:
		if err == nil {
			t.Fatal("read on a disconnected client's stream succeeded")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("open stream survived DisconnectClient")
	}
}

// TestAllowClientMayCallServer checks that the AllowClient hook can block
// and call back into the Server, including DisconnectClient, without
// deadlocking the H3 admission path.
func TestAllowClientMayCallServer(t *testing.T) {
	t.Parallel()

	dm := integration.RunDERPAndSTUN(t, mkLogger(t, "derpstun"), "127.0.0.1")
	var s *Server
	var calls atomic.Int32
	s = &Server{
		Logf:   mkLogger(t, "server"),
		Region: dm.Regions[1],
		AllowClient: func(k key.NodePublic) bool {
			calls.Add(1)
			time.Sleep(100 * time.Millisecond) // a slow lookup
			s.DisconnectClient(key.NewNode().Public())
			_ = s.Status()
			return true
		},
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	c := &Client{Server: s.TailcatAddr(), Logf: mkLogger(t, "client")}
	t.Cleanup(func() { c.Close() })
	PingForTest(t, s, c)
	if calls.Load() == 0 {
		t.Fatal("AllowClient was not consulted")
	}
	// Once admitted, the hook is not asked again.
	before := calls.Load()
	PingForTest(t, s, c)
	if after := calls.Load(); after != before {
		t.Errorf("AllowClient calls went from %d to %d for an admitted client", before, after)
	}
}

func TestDisconnectClientInvalidatesPendingApproval(t *testing.T) {
	t.Parallel()

	dm := integration.RunDERPAndSTUN(t, mkLogger(t, "derpstun"), "127.0.0.1")
	peer := key.NewNode().Public()
	disco := key.NewDisco().Public()
	var allow KeySet
	allow.Add(peer)
	var first atomic.Bool
	entered := make(chan struct{})
	resume := make(chan struct{})
	s := &Server{
		Logf:   mkLogger(t, "server"),
		Region: dm.Regions[1],
		AllowClient: func(k key.NodePublic) bool {
			allowed := allow.Contains(k)
			if first.CompareAndSwap(false, true) {
				close(entered)
				<-resume
			}
			return allowed
		},
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	result := make(chan bool, 1)
	go func() { result <- s.lb.onMeow(peer, disco) }()
	<-entered
	allow.Remove(peer)
	connected := s.DisconnectClient(peer)
	close(resume)
	if connected {
		t.Error("DisconnectClient reported an admission in progress as connected")
	}
	if <-result {
		t.Fatal("admission was granted by a revoked approval")
	}
	if _, ok := s.lb.peerConfig(peer); ok {
		t.Fatal("the revoked peer was authorized for H3")
	}
	allow.Add(peer)
	if !s.lb.onMeow(peer, disco) {
		t.Fatal("a fresh approval failed after the pending decision was invalidated")
	}
}

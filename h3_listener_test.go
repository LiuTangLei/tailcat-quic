// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package tailcat

import (
	"io"
	"net"
	"testing"
	"time"

	"tailscale.com/tstest/integration"
)

// H3 handlers normally own their connection until the callback returns. Listen
// transfers it to Accept instead: neither handoff nor closing the listener may
// close an already accepted stream, but server shutdown must still release it.
func TestH3AcceptedListenerConnectionLifetime(t *testing.T) {
	dm := integration.RunDERPAndSTUN(t, mkLogger(t, "derpstun"), "127.0.0.1")
	s := &Server{Region: dm.Regions[1], Logf: mkLogger(t, "server")}
	t.Cleanup(func() { s.Close() })
	ln, err := s.Listen(t.Context(), "tcp", ":8080")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			accepted <- c
		}
	}()
	client := &Client{Server: s.TailcatAddr(), Logf: mkLogger(t, "client")}
	t.Cleanup(func() { client.Close() })
	PingForTest(t, s, client)
	conn, err := client.DialTCPPort(t.Context(), 8080)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var serverConn net.Conn
	select {
	case serverConn = <-accepted:
	case <-time.After(5 * time.Second):
		t.Fatal("H3 connection never reached Accept")
	}
	defer serverConn.Close()
	lifetime, ok := serverConn.(interface{ Done() <-chan struct{} })
	if !ok {
		t.Fatal("accepted H3 stream has no shutdown signal")
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	_ = serverConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	const message = "accepted stream outlives listener"
	if _, err := io.WriteString(serverConn, message); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(message))
	if _, err := io.ReadFull(conn, got); err != nil || string(got) != message {
		t.Fatalf("accepted stream was closed early: %q, %v", got, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-lifetime.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("server shutdown did not release the listener's H3 callback")
	}
}

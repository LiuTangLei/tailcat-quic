// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package tailcat

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"tailscale.com/tstest/integration"
)

func TestH3SmallRoundTrips(t *testing.T) {
	dm := integration.RunDERPAndSTUN(t, func(string, ...any) {}, "127.0.0.1")
	s := &Server{Region: dm.Regions[1], Logf: func(string, ...any) {}}
	s.OnTCP = func(uint16) func(net.Conn) {
		return func(c net.Conn) { defer c.Close(); _, _ = io.Copy(c, c) }
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	client := &Client{Server: s.TailcatAddr(), Logf: func(string, ...any) {}}
	defer client.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	c, err := client.DialTCPPort(ctx, 8080)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { c.Close() })
	defer stop()

	// This is a framing/progress regression, not a throughput benchmark.
	// A single shared 25-second deadline coupled all 60 exchanges to macOS
	// race-runner scheduling. Keep every exchange bounded independently and
	// retain a total deadline; actual latency acceptance belongs to WAN tests.
	started := time.Now()
	for i := 0; i < 60; i++ {
		var sent, received [8]byte
		binary.BigEndian.PutUint64(sent[:], uint64(i))
		if err := c.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Write(sent[:]); err != nil {
			t.Fatalf("round %d write: %v", i, err)
		}
		if _, err := io.ReadFull(c, received[:]); err != nil {
			t.Fatalf("round %d read: %v", i, err)
		}
		if !bytes.Equal(sent[:], received[:]) {
			t.Fatalf("round %d: echo contents changed", i)
		}
	}
	t.Logf("60 verified round trips: %v", time.Since(started))
	if _, err := client.DiscoPing(ctx); err != nil {
		t.Fatalf("discovery after application traffic: %v", err)
	}
}

// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

// wancheck tests the actual UDP API over an isolated native H3 server.
// Numeric `tailcat serve` currently exposes TCP only, so UDP API validation
// deliberately uses this separately identified driver, not a fictional CLI API.
// Credentials travel on stdin/stdout between the private SSH supervisors and
// are never included in the client measurement report.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/tailscale/tailcat"
)

type connectionInfo struct {
	Addr string `json:"addr"`
	Port uint16 `json:"port"`
}

type probeResult struct {
	Payload  int     `json:"payload_bytes"`
	Sent     int     `json:"sent"`
	Received int     `json:"received"`
	Lost     int     `json:"lost"`
	Median   float64 `json:"median_ms"`
	P95      float64 `json:"p95_ms"`
	Max      float64 `json:"max_ms"`
}

func serve(ctx context.Context) error {
	const port = 32123 // virtual userspace port, never an OS/public listener
	s := &tailcat.Server{Logf: func(string, ...any) {}}
	defer s.Close()
	s.OnTCP = func(p uint16) func(net.Conn) {
		if p != port {
			return nil
		}
		return func(c net.Conn) {
			defer c.Close()
			_ = c.SetDeadline(time.Now().Add(30 * time.Second))
			_, _ = io.Copy(c, c)
		}
	}
	s.OnUDP = func(p uint16) func(tailcat.ConnPacketConn) {
		if p != port {
			return nil
		}
		return func(c tailcat.ConnPacketConn) {
			defer c.Close()
			buf := make([]byte, 65535)
			for {
				if err := c.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
					return
				}
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				if _, err := c.Write(buf[:n]); err != nil {
					return
				}
			}
		}
	}
	if err := s.Start(); err != nil {
		return err
	}
	if err := json.NewEncoder(os.Stdout).Encode(connectionInfo{string(s.TailcatAddr()), port}); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func probe(ctx context.Context, expectedDirect bool) (map[string]any, error) {
	var info connectionInfo
	if err := json.NewDecoder(io.LimitReader(os.Stdin, 128<<10)).Decode(&info); err != nil {
		return nil, errors.New("invalid private connection metadata")
	}
	if _, err := tailcat.ParseAddr(tailcat.Addr(info.Addr)); err != nil || info.Port == 0 {
		return nil, errors.New("invalid H3 credential or virtual service port")
	}
	c := &tailcat.Client{Server: tailcat.Addr(info.Addr), Logf: func(string, ...any) {}}
	defer c.Close()
	// Discovery ping does not establish the H3 tunnel. Warm up with a verified
	// TCP echo first, using the same Client and hence the same H3 session.
	started := time.Now()
	tcp, err := c.DialTCPPort(ctx, info.Port)
	if err != nil {
		return nil, fmt.Errorf("H3 TCP warm-up dial: %w", err)
	}
	_ = tcp.SetDeadline(time.Now().Add(30 * time.Second))
	warm := []byte("authenticated H3 UDP readiness")
	if _, err = tcp.Write(warm); err == nil {
		got := make([]byte, len(warm))
		_, err = io.ReadFull(tcp, got)
		if err == nil && !bytes.Equal(got, warm) {
			err = errors.New("warm-up integrity failure")
		}
	}
	_ = tcp.Close()
	if err != nil {
		return nil, err
	}
	ready := time.Since(started).Seconds()
	udp, err := c.DialUDPPort(ctx, info.Port)
	if err != nil {
		return nil, err
	}
	defer udp.Close()
	results := make([]probeResult, 0, 3)
	sequence := uint32(0)
	for _, size := range []int{64, 512, 1200} {
		r := probeResult{Payload: size, Sent: 20}
		rtts := make([]float64, 0, r.Sent)
		for i := 0; i < r.Sent; i++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			sequence++
			payload := bytes.Repeat([]byte{byte(sequence)}, size)
			binary.BigEndian.PutUint32(payload, sequence)
			if err := udp.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
				return nil, err
			}
			start := time.Now()
			if _, err := udp.Write(payload); err != nil {
				return nil, err
			}
			buf := make([]byte, 2048)
			for {
				n, err := udp.Read(buf)
				if err != nil {
					var ne net.Error
					if errors.As(err, &ne) && ne.Timeout() {
						r.Lost++
						break
					}
					return nil, err
				}
				if n < 4 {
					return nil, errors.New("truncated echo")
				}
				gotSeq := binary.BigEndian.Uint32(buf[:4])
				if gotSeq < sequence {
					continue
				} // late/duplicate packet; not success for this request
				if !bytes.Equal(payload, buf[:n]) {
					return nil, errors.New("UDP payload corruption")
				}
				r.Received++
				rtts = append(rtts, float64(time.Since(start))/float64(time.Millisecond))
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if len(rtts) > 0 {
			sort.Float64s(rtts)
			r.Median, r.P95, r.Max = rtts[len(rtts)/2], rtts[min(len(rtts)-1, len(rtts)*95/100)], rtts[len(rtts)-1]
		}
		results = append(results, r)
	}
	path, err := c.DiscoPing(ctx)
	if err != nil {
		return nil, fmt.Errorf("path verification: %w", err)
	}
	direct := path.Endpoint != ""
	if direct != expectedDirect {
		return nil, errors.New("unexpected H3 data path")
	}
	lost := 0
	for _, r := range results {
		lost += r.Lost
	}
	return map[string]any{"ok": lost == 0, "layer": "UDP Go API, not numeric serve CLI", "direct": direct,
		"warm_up_seconds": ready, "sent": 60, "received": 60 - lost, "lost": lost, "probes": results}, nil
}

func main() {
	mode := flag.String("mode", "client", "server or client")
	limit := flag.Duration("timeout", 180*time.Second, "total test deadline")
	direct := flag.Bool("expect-direct", true, "require a direct path")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, deadline := context.WithTimeout(ctx, *limit)
	defer deadline()
	var err error
	if *mode == "server" {
		err = serve(ctx)
	} else if *mode == "client" {
		var result map[string]any
		result, err = probe(ctx, *direct)
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(result)
		}
	} else {
		err = errors.New("unknown test mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

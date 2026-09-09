// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package tailcat

import (
	"sync"
	"testing"
	"time"

	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
	"tailscale.com/wgengine"
)

func TestH3AdmissionReclaimsDisconnectedHistory(t *testing.T) {
	now := time.Now()
	b := &locoBackend{clients: make(map[key.NodePublic]*tailcfg.Node)}
	active := key.NewNode().Public()
	lease := newH3ClientLease(now.Add(-24 * time.Hour))
	lease.update(wgengine.PeerWireGuardStateEstablished, now.Add(-24*time.Hour))
	b.clients[active] = &tailcfg.Node{Key: active}
	b.clientLeases.Store(active, lease)
	// More than two complete capacity windows must not permanently exhaust
	// an otherwise idle server just because clients use ephemeral identities.
	for i := 0; i < 3*h3MaxClients; i++ {
		if len(b.clients) >= h3MaxClients {
			retired, ok := b.retireH3ClientLocked(now)
			if !ok || retired == active {
				t.Fatal("failed to reclaim disconnected history while preserving active peer")
			}
			if _, ok := b.clients[retired]; ok {
				t.Fatal("retired identity remains authorized")
			}
			if _, ok := b.clientLeases.Load(retired); ok {
				t.Fatal("retired identity metadata leaked")
			}
		}
		peer := key.NewNode().Public()
		l := newH3ClientLease(now.Add(-2 * h3ClientGrace))
		l.update(wgengine.PeerWireGuardStateExpired, now.Add(-2*h3ClientGrace))
		b.clients[peer] = &tailcfg.Node{Key: peer}
		b.clientLeases.Store(peer, l)
	}
	if len(b.clients) > h3MaxClients {
		t.Fatal("admission state grew beyond its bound")
	}
}

func TestH3AdmissionProtectsRecentAndEstablishedSessions(t *testing.T) {
	now := time.Now()
	for _, state := range []wgengine.PeerWireGuardState{
		wgengine.PeerWireGuardStateNone, wgengine.PeerWireGuardStateHandshake,
		wgengine.PeerWireGuardStateEstablished, wgengine.PeerWireGuardStateExpired,
	} {
		lease := newH3ClientLease(now)
		lease.update(state, now)
		if _, ok := lease.reclaimable(now); ok {
			t.Fatalf("recent state %v reclaimed", state)
		}
		_, ok := lease.reclaimable(now.Add(2 * h3ClientGrace))
		if ok == (state == wgengine.PeerWireGuardStateEstablished) {
			t.Fatalf("incorrect reclamation for aged state %v", state)
		}
	}
}

func TestH3SessionCallbackDoesNotTakePolicyLock(t *testing.T) {
	peer := key.NewNode().Public()
	b := &locoBackend{}
	lease := newH3ClientLease(time.Now().Add(-time.Hour))
	b.clientLeases.Store(peer, lease)
	b.mu.Lock()
	defer b.mu.Unlock()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				b.onH3SessionState(peer, wgengine.PeerWireGuardStateEstablished)
				lease.reclaimable(time.Now())
			}
		}()
	}
	wg.Wait()
	if _, ok := lease.reclaimable(time.Now().Add(time.Hour)); ok {
		t.Fatal("established session was not protected")
	}
}

// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package tailcat

import (
	"sync/atomic"
	"time"

	"tailscale.com/types/key"
	"tailscale.com/wgengine"
)

// Keep recent admissions and transient reconnects while bounding the number
// of retained identities. Established sessions are never reclamation targets.
const h3ClientGrace = time.Minute

type h3ClientLease struct {
	// A packed timestamp and state are published together so a fresh session
	// event cannot be paired with a stale timestamp by admission cleanup.
	// Low three bits hold the session state; remaining bits are Unix seconds.
	activity atomic.Uint64
}

func newH3ClientLease(now time.Time) *h3ClientLease {
	lease := new(h3ClientLease)
	lease.update(wgengine.PeerWireGuardStateHandshake, now)
	return lease
}

func (l *h3ClientLease) update(state wgengine.PeerWireGuardState, now time.Time) {
	l.activity.Store(uint64(now.Unix())<<3 | uint64(state)&7)
}

func (l *h3ClientLease) reclaimable(now time.Time) (time.Time, bool) {
	v := l.activity.Load()
	state := wgengine.PeerWireGuardState(v & 7)
	changed := time.Unix(int64(v>>3), 0)
	if state == wgengine.PeerWireGuardStateEstablished || now.Sub(changed) < h3ClientGrace {
		return changed, false
	}
	return changed, true
}

// Called from the engine's session callback, potentially under transport locks.
// Never acquire the admission or network-policy mutex from this callback.
func (b *locoBackend) onH3SessionState(peer key.NodePublic, state wgengine.PeerWireGuardState) {
	if value, ok := b.clientLeases.Load(peer); ok {
		value.(*h3ClientLease).update(state, time.Now())
	}
}

// retireH3ClientLocked removes authorization for the oldest disconnected or
// abandoned identity. Caller holds mu and admissionMu, then must drop mu and
// ResetDevicePeer before allowing any re-admission of the retired key.
func (b *locoBackend) retireH3ClientLocked(now time.Time) (key.NodePublic, bool) {
	var candidate key.NodePublic
	var oldest time.Time
	for peer := range b.clients {
		value, ok := b.clientLeases.Load(peer)
		if !ok {
			continue
		}
		changed, reclaimable := value.(*h3ClientLease).reclaimable(now)
		if reclaimable && (candidate.IsZero() || changed.Before(oldest)) {
			candidate, oldest = peer, changed
		}
	}
	if candidate.IsZero() {
		return candidate, false
	}
	delete(b.clients, candidate)
	b.clientLeases.Delete(candidate)
	return candidate, true
}

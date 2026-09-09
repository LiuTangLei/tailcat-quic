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

// No real timestamp/state combination uses this tombstone. Retirement and a
// concurrent session callback must have a single atomic winner.
const h3LeaseRetired = ^uint64(0)

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

func (l *h3ClientLease) update(state wgengine.PeerWireGuardState, now time.Time) bool {
	next := uint64(now.Unix())<<3 | uint64(state)&7
	for {
		previous := l.activity.Load()
		if previous == h3LeaseRetired {
			return false
		}
		if l.activity.CompareAndSwap(previous, next) {
			return true
		}
	}
}

func (l *h3ClientLease) reclaimable(now time.Time) (time.Time, bool) {
	v := l.activity.Load()
	state := wgengine.PeerWireGuardState(v & 7)
	changed := time.Unix(int64(v>>3), 0)
	if v == h3LeaseRetired || state == wgengine.PeerWireGuardStateEstablished || now.Sub(changed) < h3ClientGrace {
		return changed, false
	}
	return changed, true
}

// claimRetirement rechecks the candidate at the point of retirement. A peer
// becoming established (or starting a recent handshake) between the scan and
// this CAS cannot be reclaimed. A losing callback cannot resurrect a retired
// lease. Returning false conservatively leaves admission to a later retry.
func (l *h3ClientLease) claimRetirement(now time.Time) bool {
	v := l.activity.Load()
	state := wgengine.PeerWireGuardState(v & 7)
	changed := time.Unix(int64(v>>3), 0)
	if v == h3LeaseRetired || state == wgengine.PeerWireGuardStateEstablished || now.Sub(changed) < h3ClientGrace {
		return false
	}
	return l.activity.CompareAndSwap(v, h3LeaseRetired)
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
	value, ok := b.clientLeases.Load(candidate)
	if !ok || !value.(*h3ClientLease).claimRetirement(now) {
		return key.NodePublic{}, false
	}
	delete(b.clients, candidate)
	b.clientLeases.Delete(candidate)
	return candidate, true
}

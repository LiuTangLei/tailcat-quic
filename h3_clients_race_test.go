// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

package tailcat

import (
	"testing"
	"time"

	"tailscale.com/wgengine"
)

func TestH3RetirementAndEstablishmentHaveOneWinner(t *testing.T) {
	now := time.Now()
	for i := 0; i < 2000; i++ {
		lease := newH3ClientLease(now.Add(-2 * h3ClientGrace))
		lease.update(wgengine.PeerWireGuardStateExpired, now.Add(-2*h3ClientGrace))
		start := make(chan struct{})
		updated := make(chan bool, 1)
		retired := make(chan bool, 1)
		go func() {
			<-start
			updated <- lease.update(wgengine.PeerWireGuardStateEstablished, now)
		}()
		go func() {
			<-start
			retired <- lease.claimRetirement(now)
		}()
		close(start)
		u, r := <-updated, <-retired
		if u == r {
			t.Fatalf("iteration %d: establish=%v retire=%v, want exactly one winner", i, u, r)
		}
		if r && lease.update(wgengine.PeerWireGuardStateEstablished, now) {
			t.Fatal("late callback resurrected a retired lease")
		}
		if u {
			if _, ok := lease.reclaimable(now.Add(time.Hour)); ok {
				t.Fatal("winning established session became reclaimable")
			}
		}
	}
}

func TestH3RetirementRechecksStateAfterCandidateScan(t *testing.T) {
	now := time.Now()
	lease := newH3ClientLease(now.Add(-2 * h3ClientGrace))
	if _, ok := lease.reclaimable(now); !ok {
		t.Fatal("expected abandoned candidate")
	}
	lease.update(wgengine.PeerWireGuardStateEstablished, now)
	if lease.claimRetirement(now) {
		t.Fatal("candidate scan was allowed to erase a newly established session")
	}
}

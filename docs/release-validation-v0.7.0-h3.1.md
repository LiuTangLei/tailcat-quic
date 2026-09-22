# Tailcat v0.7.0-h3.1 release validation

Candidate status: application integration complete; WAN comparison and packaged
release gates are pending. Do not publish solely on this candidate document.

## Exact source scope

This merge includes upstream Tailcat v0.7.0 at
`15ab9e68bfc6534a61797d7af28cedd42b54a3a5`. The application remains in
LiuTangLei/tailcat-quic, with no new mirror repository. Shared QUIC is pinned
at `v0.63.0-tailscale.1`; the Tailscale integration library is pinned at
`v1.102.5-0.20260922163518-49de27a3a715`. Both are publicly downloadable.

The library carries the existing MSS, batching and lifecycle work, selected
upstream Android DNS/CA/interface support, and the upstream gVisor CUBIC/RACK
recovery change with its matching September 15 gVisor version. It does not
claim to merge all of upstream Tailscale main. Native-TUN ready reads are not
a direct optimization of Tailcat's user-space TCP streams.

The new Server.Listen API is integrated into the authenticated H3 TCP path:
listener precedence, post-Accept ownership and shutdown are verified rather
than assuming the upstream netstack-only dispatcher is sufficient. TCP payload
remains in real reliable H3 streams; UDP uses authenticated DATAGRAM transport.
No encryption, node authorization, connection secret or congestion gate is
relaxed. Browser-equivalence or universal censorship resistance is not claimed.

## Local checks completed

Full application tests and vet passed using public immutable dependency pins.
Listener lifetime and precedence race tests passed three times. Shared-library
stream, netstack and network-monitor race tests passed. Android compatibility
unit test binaries were cross-compiled and executed on Linux amd64; this is
not a physical Android-device test.

A local fixture initially exposed post-test home-router portmapper logging.
The test-only setup now explicitly disables external portmapper probing, as
intended by the upstream hermetic fixture; this does not change application
runtime or release transport policy.

## Why SG-to-US is not the release optimization target

Earlier same-host native-WG diagnostic measurements also showed low throughput
and heavy retransmission. Their failed integrity probes are preserved and are
not counted as acceptance passes. A separate completed 3x15-second WG run
measured SG-to-US at 159.27 / 183.43 / 180.05 Mbps, with substantial retransmits.
This does not prove WG reproduced QUIC's exact single-digit failure mechanism.
It justifies moving this release's performance comparison to AU/US, as requested.

## Required remaining release evidence

Compare the actual old v0.6.0-h3.3 and new release-profile CLI on the same WAN
pair; preserve every completed throughput sample and functional failure.
Verify direct outer endpoints, integrity, idle recovery and cleanup. The
existing probe excludes two initial seconds from iperf measurement; report
that warmup rather than presenting it as an unomitted kernel-TUN benchmark.
Then pass the repository's cross-platform tests, package checksums and actual
packaged-client verification before publishing the draft.

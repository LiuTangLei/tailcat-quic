# Tailcat H3 v0.6.0-h3.2 — performance and validation

Measurements ran on September 9, 2026 UTC (September 10 locally). Two authorized Linux amd64 hosts are anonymized as A and B. Their addresses, connection codes and credentials are deliberately excluded. Finite tests are not a guarantee that all defects or network limitations have been eliminated.

## What changed

TCP application connections now use separate reliable HTTP/3 CONNECT streams on the existing authenticated QUIC connection. They no longer traverse a second, user-space TCP connection inside unreliable IP DATAGRAMs. UDP and IP datagram APIs continue to use CONNECT-IP / QUIC DATAGRAM on the same H3 data plane. BBRv3 remains selected at both QUIC endpoints; there is no WG or AWG data-plane fallback.

A plain TLS connection is not sufficient to open TCP CONNECT streams. The implementation finds the already authenticated CONNECT-IP session for that exact QUIC connection, checks the current node authorization, and dispatches only through the embedding application's allowed TCP ports/forward handlers. Client addresses presented to SSH are derived from authenticated node identities, not supplied request headers.

Other changes:

- Eight lost/incomplete fragment assemblies no longer block all new fragmented packets until their two-second timeout. The oldest incomplete assembly is replaced under pressure, with the same eight-assembly memory bound. Overlapping/malformed fragments still fail closed.
- Optional MSS capping avoids fragmenting ordinary TCP packets retained on the IP path. It does not lower the IPv6 MTU or change UDP semantics. The new TCP stream path does not rely on this cap.
- TCP read/write deadlines operate on bounded, owned queues, not in the middle of HTTP/3 frame headers. Timed-out I/O can be resumed without corrupting DATA framing.
- Half-close drains accepted bytes before FIN. One-shot clients wait for actual stream-byte/FIN acknowledgment before terminating; local Close is not mistaken for peer acknowledgment.
- Transport shutdown sends CONNECTION_CLOSE before closing magicsock. Blocked readers, full receive queues and unrelated application callbacks do not hold the transport shutdown hostage.
- Long-lived SSH/file streams are not truncated by the IP-only transport's periodic full QUIC connection replacement. These embedded stream sessions use QUIC packet-key updates and live node revocation. Diagnostics report that distinction rather than inventing a full-handshake renewal timer.

## Same-topology official WG comparison

The baseline is the **actual official tailcat v0.6.0 Linux amd64 release executable**, not bare TCP and not a locally modified WG build. Its downloaded archive matched the published SHA-256:

`f3597a9ad02f5cca538f8f5a6f89123910bce3e9611d1e5a8e96d5f2d3cc90fd`

All throughput rows use receiver-side iperf3 results, 15-second measurements and two omitted startup seconds. Local echo probes run concurrently during load. Four streams means four application connections sharing one H3 QUIC connection, not four independent outer tunnels. Both directions, roles and complete-data integrity were exercised. Runs are sequential, not a simultaneous controlled laboratory comparison.

### A as server, B as client

| Direction / streams | Official WG run 1 | Official WG run 2 | New H3 candidate |
| --- | ---: | ---: | ---: |
| B → A / 1 | 110.18 Mbps | 57.08 Mbps | **308.41 Mbps** |
| A → B / 1 | 50.51 Mbps | 30.47 Mbps | **161.96 Mbps** |
| B → A / 4 | 173.92 Mbps | 98.26 Mbps | **244.25 Mbps** |
| A → B / 4 | 111.56 Mbps | 102.86 Mbps | **168.63 Mbps** |

The pre-optimization H3 native-IP baseline under this same test procedure delivered 41.49 / 34.43 Mbps in the two single-stream directions and 30.78 / 39.08 Mbps with four streams. Fragment and MSS repairs alone did not establish WG parity; temporary BBRv1 and CUBIC diagnostic builds did not consistently solve the problem either. Those controllers are not substituted for the release default.

The listed new candidate executable's SHA-256 is:

`2cf6f0d33f6da3bd1bb2b9255672a8ef78313545bf9356fd99e226ee97184b6d`

It was an instrumented development build. Safe per-second counters demonstrated BBRv3, HTTP/3 TCP stream byte traffic, zero WireGuard devices, and no local raw/datagram-queue drops. The final source includes additional shutdown/read-pump cleanup and diagnostic clarifications after that performance build; packaged-release checks are recorded separately, rather than falsely calling a development binary a downloaded Release artifact.

### Roles reversed

An earlier reliable-stream candidate with B as server and A as client measured:

| Direction | One stream | Four streams |
| --- | ---: | ---: |
| A → B | 183.63 Mbps | 189.90 Mbps |
| B → A | 288.41 Mbps | 355.24 Mbps |

Another candidate with the original roles measured 293.10 / 180.57 Mbps single-stream and 162.41 / 194.65 Mbps with four streams. These repetitions support an improvement on this tested path, not a universal speed guarantee or a claim of congestion-control superiority on every network.

## Integrity, latency and resources

The latest direct candidate checked **34,079,744 bytes** with SHA-256: a 1 KiB warm-up, 12 MiB in each direction, four concurrent 2 MiB transfers split across directions, and transfers resumed after six seconds idle. All hashes matched. Short transfers still include connection/flow-control startup: the complete 12 MiB transfers measured 35.67 and 48.32 Mbps, not the peak steady-stream rates above.

Idle TCP echo median was **171.83 ms**. Loaded medians were **174.34–194.76 ms**. Loaded p95 ranged **208.43–675.87 ms**, with one maximum **1045.22 ms**. Throughput improved substantially, but this does **not** establish parity with WG's loaded latency, eliminate all stalls, or justify presenting only the median as the worst-case experience.

The candidate's sampled peak RSS was 88.2 MiB on B/client and 58.8 MiB on A/server. Mean CPU across the whole run, including idle phases, was about 53.2% and 54.3% of one core; short sampled peaks were 126% and 124%. These are process observations, not a minimum hardware sizing guarantee.

## UDP and DERP

UDP is verified by the separate `internal/wancheck` driver using actual `OnUDP` / `DialUDPPort` APIs. Numeric `tailcat serve PORT` retains the upstream TCP-only CLI semantics; this report does not mislabel library UDP coverage as an implicit UDP CLI listener.

Each UDP run sends 20 sequence-checked echoes at each of 64, 512 and 1200 bytes, after establishing real authenticated application data (not merely a discovery ping). The latest direct and forced-DERP runs each returned **60/60** payloads correctly. This is an integrity/echo sample, not sustained high-rate UDP loss certification.

With both endpoints explicitly forced through DERP, the candidate measured **23.03 Mbps B → A** and **22.98 Mbps A → B**, in short five-second receiver-side measurements. It additionally verified **3,671,040 bytes** with SHA-256 across both directions, concurrency and idle recovery. Both endpoint logs confirmed forced relay, no direct path and no WG device. UDP separately reported `direct=false` and passed 60/60 echoes.

Relay idle echo median was 170.26 ms. Loaded medians rose to 238.72–292.11 ms, with a maximum 519.24 ms. Relay performance depends on the relay and its outer transport; it is not covered by the direct-path speed claim.

## Regression evidence

Local validation used the actual edited source and dependencies, not test stubs:

```sh
go test -count=1 -timeout=120s ./...
go test -race -p=1 -parallel=1 -count=1 -timeout=180s ./...
go vet ./...
go test -race -p=1 -parallel=1 -count=1 -timeout=180s \
  tailscale.com/wgengine/wgtransport/quicbind \
  tailscale.com/wgengine/quicip tailscale.com/wgengine
```

TCP regressions cover public-TLS-without-node-auth denial, wrong application credentials, denied service ports, simultaneous streams, half-close, complete EOF/FIN drain, source identity, long-lived sessions, peer revocation, cancellation, server shutdown, concurrent use and `golang.org/x/net/nettest` connection/deadline contracts. A separate full-read-queue shutdown regression is repeated five times under the race detector. Fragment-pressure, checksum/MSS and malformed-option tests remain applicable to the datagram path.

The QUIC dependency passed its full ordinary suite and `go vet ./...`; the new acknowledgment behavior and BBRv3 tests were also run with the race detector. The H3 and tailcat suites passed locally under race instrumentation. The unserved-port test now waits for an explicit authenticated denial instead of using a 100 ms cancellation that could race the denial response under instrumentation; the permission check itself was not removed.

## Reproduction and deployment boundary

`host_pair_probe.py` supports the official-WG baseline and H3 candidate under the same procedure, accepts SSH destinations only at runtime, and writes a new JSON evidence file with `--report`. `tailcat_perf` is an explicit diagnostic build tag; ordinary release builds do not emit periodic diagnostic snapshots. The corresponding per-run JSON files are local test evidence, not a place to store public credentials.

No production VPN, route, firewall, operating-system congestion setting or existing service was changed on the two hosts. Temporary fixtures, listeners and executable copies are removed after the tests. Both peers should upgrade together for the reliable TCP stream capability; an older H3 peer that does not advertise it is rejected explicitly rather than silently starting WG/AWG.

This report establishes the speed improvement and the tested functional/security behavior on the specified finite matrix. It does not certify all NATs, all target architectures, very long production uptime, browser indistinguishability, active-probing resistance, or zero bugs. A GitHub Release must still pass its packaging/CI gates and packaged-asset verification before publication.

# tailcat-quic v0.7.0-quic.2 validation — 2026-09-23

## Scope and naming

This 0.7 release retains the upstream v0.7.0 application merge and QUIC/HTTP3-only
transport. New tags use `quic.N`, never `h3.N`. Historical unshipped candidate tags
are not rewritten. GitHub Actions is disabled for tailcat-quic and the shared
quic-go fork; both current development trees omit .github. Packaging is local,
with explicit executable verification and manual Release publication.

The earlier final tag workflow failed in a macOS race-instrumented TestPing
subprocess timeout after printing a relay pong, not a compiler error or a reported
data race. Eight local repetitions of that test with the race detector passed.
This is not claimed as a complete root-cause proof or a reason to omit the test.

## Retained optimization

The H3 TCP read pump formerly allocated a fresh 32 KiB slice for every stream
read. It now reuses explicitly owned buffers from a backend cache capped at 64
idle blocks (2 MiB). Used bytes are cleared before reuse across streams. Active
storage remains limited by the existing two-entry read queue and pump/current
read slots; the cache cap is not a cap on an entire application's memory.

An application Read additionally collects already-ready chunks into its available
buffer without waiting for more. This reduces small handoffs and subsequent
socket writes without a coalescing timer. The previous bounded graceful close,
FIN acknowledgement, explicit abort and authorization semantics remain. No
cryptography, congestion algorithm, flow-control window, route or OS setting was
changed to produce the measurements below.

The implementation adapts part of the earlier unshipped read-pool experiment;
its old immediate full-close behavior and unrelated socket/cache patches were
not imported. CPU/GC allocation improvement is not a promise of lower peak RSS
on every workload.

## Exact dependency pins

| Component | Public immutable source |
| --- | --- |
| Tailscale H3 integration | `github.com/LiuTangLei/tailscale v1.102.5-0.20260923003659-b4f1aa3cd300` |
| Shared QUIC | `github.com/LiuTangLei/quic-go v0.63.0-quic.2` (`1242e7347f2d15a6fd0b5b8ccbb6e7a5318f4608`) |
| WG/TUN primitives | `github.com/LiuTangLei/wireguard-go v0.0.33-0.20260910045057-ed22747d204e` |
| gVisor | `v0.0.0-20260915211658-a6f909f08a72` |

The QUIC library tag differs from the prior .63 tag only in repository guidance
and removal of GitHub automation, not runtime protocol code. Release builds use
these public module downloads, not local replacements. The shared Tailscale pin
is a library dependency, not a rollout of new production VPN daemons.

## Alternating WAN comparison

AU is the server and US1420 is the client. Four complete runs alternate old/new
then new/old to reduce order bias. The baseline is the immediately preceding
0.7 candidate (872d3c108 application with 0df273162 integration), not the original
0.6 release. Both sides use normal release build tags, no profiling or diagnostic
build tag, Go 1.27.1, identical authentication, BBRv3 and service setup. Only the
candidate has the read-buffer/coalescing change.

Each run tests one and four TCP streams in both directions: ten measured seconds
plus two explicitly omitted warmup seconds. Both sides' direct public endpoints
were observed. Management SSH uses existing private connectivity; throughput
does not pass through that management connection. All complete samples, including
slower ones, are retained.

| Data direction | Streams | Baseline rounds (Mbps) | Candidate rounds (Mbps) | Baseline mean | Candidate mean |
| --- | ---: | --- | --- | ---: | ---: |
| US1420 to AU | 1 | 312.23 / 286.48 | 299.91 / 298.90 | 299.36 | 299.41 |
| US1420 to AU | 4 | 328.94 / 337.88 | 370.85 / 366.08 | 333.41 | 368.47 |
| AU to US1420 | 1 | 255.10 / 207.58 | 259.23 / 266.48 | 231.34 | 262.86 |
| AU to US1420 | 4 | 195.87 / 227.24 | 297.79 / 313.93 | 211.56 | 305.86 |

The two four-stream means improve about 10.5% and 44.6%. The forward single-stream
mean is essentially unchanged, and the reverse single-stream mean improves about
13.6%. There are only two independent runs per build: these are observed samples,
not confidence bounds or an all-network speed guarantee. Earlier historical
measurements are not substituted into this table.

All four runs passed sequential/concurrent SHA-256 content checks, idle recovery,
required H3-only engine and direct-path evidence. All loaded TCP echo samples
returned without an application echo error. This is not a measurement of zero
packet loss or a guarantee of low tail latency. Every run recorded successful
cleanup. The comparison used test builds with local dependency source overrides;
actual public-pin release executables are verified separately before publication.

Comparison Linux executable hashes:
- Baseline: `4f744698bcab686f334309ccf05fd6fcd411ab36a49784039b68160e792df97f`
- Candidate: `221b961e353923c5d052625f28f364f31b1138fcc599a2e8ae47222011a87861`

## Correctness and allocation checks

The full quicbind suite passed. Targeted race tests ran three times; read-pool
and ready-read cases ran 20 times. Coverage includes cache cap/clearing, partial
reads surviving deadlines, EOF with final bytes, no waiting to fill a buffer,
concurrent close with a full queue, and 100 EOF/graceful-close interleavings per
case invocation. Existing authentication, revocation, final-byte delivery and
explicit reset/close budget tests remain.

The full Tailcat application/CLI suite and serialized race suite passed against
the candidate. Local buffer microbenchmarks reduced warmed per-cycle allocation
from 32792 B/op to 0 B/op; that is an allocation diagnostic, not network speed.

## Release verification and limits

Local scripts require public pins and a clean checkout for packaging. Seven
Linux/macOS/Windows executable archives plus three deb and three rpm packages
are generated with checksums. Native executable tests, final direct/relay/UDP
checks and upload readback are reported with the Release's separate validation
record; source-level tests alone are not called package validation.

Native Windows/Android device execution and prolonged multi-hour load are not
claimed by this source report. The old SG/US instability is not claimed fixed:
this optimization was evaluated on AU/US, as requested after the WG control
also showed path-dependent performance. Production VPN services are not changed
by building or testing these isolated CLI executables.

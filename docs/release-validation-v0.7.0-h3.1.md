# Tailcat v0.7.0-h3.1 release validation

This document records pre-tag source and WAN evidence. The tag's Release
workflow additionally gates publication on Linux/macOS/Windows tests, all
cross-builds, package SHA-256 checks and end-to-end tests using the actual
packaged native CLI. GoReleaser creates a draft; that draft must not be
published until the packaged-client jobs succeed.

## Sources and scope

The application merges official Tailcat v0.7.0 at
`15ab9e68bfc6534a61797d7af28cedd42b54a3a5`. It remains exclusively in
LiuTangLei/tailcat-quic. Public immutable shared dependencies are:

- QUIC: `LiuTangLei/quic-go v0.63.0-tailscale.1` / `658bc72e5bcd`.
- Integration library: `LiuTangLei/tailscale
  v1.102.5-0.20260922163518-49de27a3a715`.
- Shared WG/TUN primitives: `v0.0.33-0.20260910045057-ed22747d204e`.
- gVisor: `v0.0.0-20260915211658-a6f909f08a72`; compiler: Go 1.27.1.

No tailcat-tailscale or tailcat-quic-go mirror is required. The integration
library selectively backports official Android DNS/system-CA/interface
support and gVisor CUBIC/RACK recovery with Windows clock quantization handling;
it does not claim to merge all of upstream Tailscale main. Outer QUIC still
uses BBRv3. CUBIC/RACK is the inner user-space netstack policy, not a replacement
for QUIC's congestion controller.

The existing MSS, bounded DATAGRAM batch/direct-receive and lifecycle work is
retained. Kernel TUN ready-read batching is not a direct acceleration of
Tailcat's reliable TCP stream path, so Tailscale VPN speed figures are not
reused as Tailcat results. TCP forwarding remains authenticated reliable H3
streams, not nested TCP-over-DATAGRAM. UDP remains unreliable QUIC DATAGRAM.

## 0.7 compatibility fixes

Upstream's Server.Listen dispatcher was adapted to the direct H3 TCP path.
Explicit listener ports take precedence over OnTCP, and Accept transfers
connection ownership without triggering the backend callback's deferred Close.
Closing a listener does not close accepted streams; closing the server releases
them. Tests cover those contracts through actual local H3, not mock sockets.

A new upstream SSH/DNS exposure probe allocated a temporary tunnel client but
only closed its SSH stream. The CLI now closes that client on all return paths,
including denied handshakes/timeouts. This fixed an observed post-test DERP
writer leak; the complete race suite subsequently passed.

The upstream test's intended hermetic setup is preserved explicitly for the
older H3-compatible library: local fixture tests disable external home-router
portmapper probing. This is test-only and does not alter runtime security or
release network policy. No encryption, current-peer authorization, mandatory
connection secret, or congestion gate was disabled to obtain these results.

## Same-host WAN comparison

AU was the server and US1420 the client. The US machine has one vCPU. SSH
management used the existing private management network; recorded outer data
endpoints were the two machines' public addresses, not overlay addresses.

The old executable was built from the exact `v0.6.0-h3.3` tag (`3b4b6bf5d`).
The candidate was built from integration commit `c1c7d3205`. Both used Go 1.27.1,
CGO disabled, their own release-profile build tags and the actual CLI's
serve/forward path. Order was old/new, then new/old. This is two alternating
rounds, not a large randomized statistical study.

iperf measures 10 seconds after an explicitly omitted 2-second warmup. One
and four inner TCP connections are measured in both directions. Loaded RTT
uses verified application TCP echo alongside the transfer, not management
SSH or an outer ICMP ping. There is no artificial aggregate bitrate limit.
These are Tailcat stream-proxy results, not kernel-TUN/WG parity benchmarks.

| Direction | Streams | Old samples, Mbps | Candidate samples, Mbps | Old mean | Candidate mean |
|---|---:|---|---|---:|---:|
| US1420 to AU | 1 | 302.53 / 305.51 | 274.76 / 300.16 | 304.02 | 287.46 |
| US1420 to AU | 4 | 329.37 / 326.94 | 363.22 / 345.08 | 328.16 | 354.15 |
| AU to US1420 | 1 | 216.10 / 232.31 | 225.96 / 224.96 | 224.21 | 225.46 |
| AU to US1420 | 4 | 188.26 / 215.40 | 174.10 / 234.93 | 201.83 | 204.52 |

US-to-AU four-stream mean increased about 7.9%, but single-stream mean decreased
about 5.4%. The reverse means were nearly unchanged. The reverse four-stream
candidate also ranged from 174.10 to 234.93 Mbps; no low complete sample was
removed. This is not a universal speed-up or an all-direction performance SLA.

For US-to-AU four streams, loaded-echo p95 was 354/303 ms on the old version
and 243/250 ms on the candidate. Other directions were mixed; this does not
prove stable loaded tail latency. Candidate client peak RSS was 76.4/79.7 MiB,
versus 85.8/82.7 MiB for the baseline. Process CPU means were similar. CPU/RSS
were sampled across the whole fixture, not isolated per transport operation.

All four complete fixtures passed sequential/concurrent SHA-256 content checks,
6-second idle recovery, both-endpoint H3/no-WG engine checks and owned cleanup.
Each verified 25,691,136 application bytes in addition to iperf payload. Loaded
echo samples in these complete fixtures had no reported errors. Short sampling
is not evidence of zero long-term loss, zero leaks or no tail-latency spikes.

## Final runtime recheck and UDP

After the SSH-probe resource fix (`711d88b7e`), another release-profile CLI
fixture ran on the same AU/US pair. With 8 measured seconds and the same
2-second warmup, US-to-AU P1/P4 received 308.72/330.02 Mbps; AU-to-US P1/P4
received 233.96/236.62 Mbps. These differently timed samples are kept separate
from the two-round table. Source edits made after that runtime fix concern
only the probe report fields, package verifier and release documentation.

The separately identified Go UDP-API driver sent 20 payloads at each of
64, 512 and 1200 bytes. All 60 echoes returned with matching content and a
direct-path result. This is an API integrity/latency test, not a high-rate UDP
throughput result or a claim that numeric `serve PORT` enables arbitrary UDP.
The 0.7 exit-node UDP CLI behavior is covered by CLI end-to-end tests.

The final fixture also passed sequential/concurrent hashes, idle recovery and
cleanup. Observed loaded echo reached about 399 ms in that run. Long-duration
soak, diverse NATs, high-rate UDP and real Android devices are not covered here.

## Forced relay smoke test

The post-fix CLI was also run with DERP forced on both endpoints. Both logs
reported forced relay and no direct endpoint; H3/no-WG checks, hashes,
concurrent transfers, idle recovery and cleanup passed. With three measured
seconds after two seconds of warmup, P1 throughput was 23.82/22.08 Mbps.
Loaded echo reached 1235 ms, so this is a functionality smoke test, not a
relay-latency guarantee. UDP was not measured in this forced-relay fixture.

## WG and SG scope

Earlier same-host native-WG diagnostics also showed low throughput and heavy
retransmission. Their failed integrity probes are not acceptance passes. A
separate completed 3x15-second WG run measured SG-to-US at
159.27 / 183.43 / 180.05 Mbps with substantial retransmits. This does not prove
WG reproduced QUIC's exact single-digit failure mechanism. Following the
requested scope, this release does not attempt to solve that SG/US path.

## Local verification and publication gates

Public dependency download, go mod tidy/verify, complete application tests,
vet and the full serialized race suite passed locally. Listener-specific race
checks passed three times. The shared library's stream, netstack and monitor
race tests passed. Android DNS/runtime support unit-test binaries were executed
on Linux amd64; physical Android behavior is not claimed. Release build tags
are checked against the actual selected feature set, not a stale copied list.

Pre-tag CI run `35756462078` on runtime commit `711d88b7e` passed all
Linux, macOS and Windows full tests/race checks, dependency tidy/portability,
and all seven cross-builds. Subsequent source edits were only reports and
package/probe verification scripts. The final tag's Release run repeats the
gates and verifies the actual artifacts.

The workflow checks all seven executable targets and creates Linux deb/rpm
packages. Native Linux/macOS/Windows jobs verify the downloaded archive and
binary version, immutable dependency versions, clean VCS revision, documentation
and checksums, then run CLI end-to-end tests with TAILCAT_TEST_BINARY pointing
at the extracted executable. See the Release workflow for authoritative results
on the final tag; a successful local build alone is not publication approval.

## Private evidence and reproducibility

Raw WAN reports remain under the Mac audit directory
`tailscale-all/audits/tailcat-07-20260922/` and are not committed, because they
contain infrastructure endpoints. Public summaries use aliases only.

- `au-us/driver.json`: finite alternating comparison and each cleanup status.
- `au-us/baseline-{1,2}.json`, `candidate-{1,2}.json`: all 16 paired samples.
- `final-check-au-us.json`: post-fix runtime/UDP recheck and cleanup.
- `relay-check-au-us.json`: independent forced-DERP smoke test and cleanup.
- Baseline CLI SHA-256: `eb4e76b5c811c509a9434b0017d597cca74c446667e6933e732873ba537dfcd1`.
- Paired candidate SHA-256: `6f887a5f2693820fd6a5476bdee6a1b8e466af84b2b68fbdc204fbff1828e041`.
- Final runtime-check SHA-256: `0166f3910e538f56bd148e33d7d7dccf4779cac0df6e1a3daa129db29b25a598`.

Release archives have their own checksums and clean final-tag build identity;
benchmark executable hashes are not substituted for package verification.
Production tailscaled instances are not deployment targets of this release.

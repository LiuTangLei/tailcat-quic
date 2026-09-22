# Tailcat v0.7.0-h3.2 release validation

**Release channel: prerelease.** Directional WAN performance regressions remain;
this version does not replace the existing stable release or claim an
all-direction throughput improvement.

This is the successor to the unpublished h3.1 candidate. It retains the full
upstream v0.7.0 application merge, shared QUIC 0.63, BBRv3, listener ownership,
Android support and other integration changes described in the h3.1 report.
Its additional runtime fix is bounded receive-side draining at normal stream
Close. Publication still requires the final tag's full platform and actual
packaged-executable workflow gates.

## Why h3.1 was not published

The runtime branch passed Linux/macOS/Windows CI (`35756462078`). However the
final-tag Release run `35758516988` reproduced a macOS SSH forced-command
shutdown race: the command response arrived, then the proxy printed a stream
reset error while waiting for final bytes/FIN acknowledgements. The release
workflow was stopped; no public h3.1 release was created. The existing tag is
not rewritten or deleted, and the failed gate remains part of the evidence.

The server's full connection Close sent STOP_SENDING immediately, racing the
client's final protocol disconnect and FIN. A reset is not acknowledged data,
so ignoring reset error code zero would have hidden a potential delivery error.
The implementation instead lets the existing receiver finish the peer's final
frames after closing the application side, bounded by five seconds and one MiB
of additional payload. It reuses the existing buffer and reader, does not add a
queue or a per-close goroutine, and checks live authorization. Explicit
CloseRead still cancels immediately. Budget exhaustion, malformed frames,
timeout and peer cancellation still cause errors; the strict write/FIN-ACK
requirement is unchanged.

## Immutable dependencies

```
tailscale.com => github.com/LiuTangLei/tailscale v1.102.5-0.20260922172630-0df273162480
github.com/quic-go/quic-go => github.com/LiuTangLei/quic-go v0.63.0-tailscale.1
```

Shared WG/TUN primitives remain `v0.0.33-0.20260910045057-ed22747d204e`,
gVisor remains `v0.0.0-20260915211658-a6f909f08a72`, and builds use Go 1.27.1.
There are no local filesystem replacements or new library mirrors.

## Targeted regression evidence

The previously failing real SSH forced-command test passed 40 repetitions.
SSH/exec/exposure-probe race checks passed three repetitions. Real H3 tests
cover full response and trailer integrity after peer response EOF, preservation
of explicit-cancel failure, deadline cleanup of a silent peer and byte-budget
cleanup of a continuing peer. The shared H3 full suite and vet passed. This is
bounded test evidence, not proof of all possible timing schedules or long-run
leak freedom.

## Performance baseline and limits

The earlier AU/US alternating comparison (old/new, new/old) measured the same
0.7 runtime except for the close-drain fix. Each transfer had ten measured
seconds and two explicitly omitted warmup seconds. All original samples are
retained in the h3.1 report; they are not relabeled as h3.2 binary samples.

| Direction | Streams | Old 0.6.0-h3.3 mean Mbps | Earlier 0.7 candidate mean Mbps |
|---|---:|---:|---:|
| US1420 to AU | 1 | 304.02 | 287.46 |
| US1420 to AU | 4 | 328.16 | 354.15 |
| AU to US1420 | 1 | 224.21 | 225.46 |
| AU to US1420 | 4 | 201.83 | 204.52 |

Thus the original four-stream US-to-AU improvement was about 7.9%, but its
single-stream mean declined about 5.4%. Reverse averages were nearly unchanged.
These small-sample WAN differences are not a general performance guarantee.
Tailscale IP-tunnel batch improvements are not claimed as a direct speed gain
for Tailcat's reliable-stream forwarding. The release uses no encryption or
congestion-control bypass.

## Post-close-fix h3.2 comparison

The actual h3.2 runtime commit `ddacde0ea` was then rebuilt with published pins
and release flags. On the same AU-server/US-client pair, another old/new,
new/old comparison completed all four fixtures with cleanup_errors empty.
Each transfer used ten measured seconds plus two omitted warmup seconds.
Both ends reported H3/no-WG and public direct endpoints, not a nested VPN data
path. All loaded echo probes and sequential/concurrent SHA-256 checks passed.

| Direction | Streams | Old samples Mbps | h3.2 samples Mbps | Old mean | h3.2 mean |
|---|---:|---|---|---:|---:|
| US1420 to AU | 1 | 285.92 / 287.42 | 300.38 / 308.94 | 286.67 | 304.66 |
| US1420 to AU | 4 | 332.67 / 358.33 | 344.84 / 374.00 | 345.50 | 359.42 |
| AU to US1420 | 1 | 222.27 / 229.32 | 244.99 / 153.01 | 225.80 | 199.00 |
| AU to US1420 | 4 | 227.72 / 239.77 | 192.64 / 155.93 | 233.75 | 174.29 |

Forward means improved about 6.3% and 4.0%, but reverse means declined about
11.9% and 25.4%. Those complete slow samples are retained. The small alternating
experiment cannot prove all variance is caused by the code, but it does not
justify a stable performance replacement. Publication is therefore explicitly
as a prerelease, leaving the earlier stable version available.

An earlier separate h3.2 fixture recorded 279.57/362.37 Mbps for forward P1/P4
and 146.60/133.52 Mbps for reverse P1/P4. It is also retained, not discarded.
Its 64/512/1200-byte UDP API echo test passed all 60 payloads. Every completed
fixture passed content integrity and six-second idle recovery. These do not
establish high-rate UDP or zero-loss guarantees. Short-lived receive draining
uses bounded additional work; these small runs do not prove long-run memory
behavior. Candidate whole-fixture RSS observations were 79.8/74.1 MiB on US
and 91.1/70.9 MiB on AU, not uniformly below the baseline.

Raw evidence: `h3.2-comparison/{baseline,candidate}-{1,2}.json`, its finite
`driver.json`, and `h3.2-check-au-us.json` in the private audit directory.
The tested h3.2 CLI SHA-256 is
`41efb23fdf3336cb2e221824ed20721cf0aaf70aa6146e67de3fe054c020e629`.
The baseline SHA-256 is unchanged from the preceding report. Subsequent source
changes are release documentation and verification only; the tagged artifacts
must pass their own clean-build identity and packaged CLI gates.

The original forced-DERP smoke in the h3.1 report precedes the close fix and
is not relabeled as a h3.2 WAN result. Local forced-relay/CLI regressions run
through the full suite. No all-NAT, high-rate UDP, long-duration soak, physical
Android or identical browser-fingerprint claim is made. Full/packaged CI is
mandatory, not replaced by successful cross-compilation.

# Tailcat v0.7.0-h3.2 release validation

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

The post-close-fix CLI is separately tested before tagging; its exact results
and release workflow outcome must be recorded before publication. Original
sequential/concurrent hashes, idle recovery, 60/60 UDP API echoes and forced
DERP checks are detailed in the earlier report, with their binary identities.
No all-NAT, high-rate UDP, long-duration soak, physical Android or identical
browser-fingerprint claim is made. Full/packaged CI is mandatory, not replaced
by successful cross-compilation.

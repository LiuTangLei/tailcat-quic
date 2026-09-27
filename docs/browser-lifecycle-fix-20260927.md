# Browser lifecycle fixes and validation — 2026-09-27

Base: application `v0.7.0-quic.3`, commit
`4ac6137cbce9af2bae62095cd68dc7bc0b44842f`. Changes are on
`fix/browser-lifecycle-quic4-20260927`; this is not a published release.

## Fixes

Promise executors now release their Go/JS registration immediately after the
Promise constructor invokes them. Their asynchronous operation retains its own
references until settlement. Completed writes no longer keep every payload alive.

Connection and listener methods bind to a shared dispatcher. Closing an owner
removes its Go callback references. Saved JavaScript methods remain safe:
repeated close is a no-op, and I/O after close rejects. In-flight operations retain
their resources until they settle. A connection's existing Done signal also
releases its owner on session shutdown. Client/server teardown runs outside the
JavaScript callback so network cleanup can continue through the event loop.

The browser sender closes its connection in a finally block, including file-read,
write, half-close and response-read failures. Successful completion still requires
the peer's EOF; an error does not set sendDone or become successful delivery.

Transport, authentication, FIN/ACK semantics, congestion policy and public module
pins are unchanged. Both native peers continue using reliable H3 streams for TCP,
DATAGRAM for UDP, mandatory secrets/node authentication and BBRv3.

## Regression checks

Run the production bridge in the Go JS/WASM runtime, then test failure cleanup:

```sh
GOOS=js GOARCH=wasm go test \
  -exec="$(go env GOROOT)/lib/wasm/go_js_wasm_exec" -v \
  web/bridge_js.go web/bridge_js_test.go
node --test web/send_cleanup_test.mjs
go test ./web -run-headless-browser-tests -count=1 -timeout=8m
```

The bridge test sends 32 MiB and checks retained Go heap after GC, outstanding
read/write cancellation, session shutdown, half-close, saved methods, repeated
close and 100 connection/listener lifecycles. The original write path retained
about 32 MiB; the fixed path retained no additional payload-sized Go heap in the
local run. This is not a browser RSS or multi-hour memory-soak claim.

Six injected sender paths cover success and failures at dial, file read, write,
half-close and response read. Every acquired connection closes exactly once;
the original error is preserved and failures never mark successful delivery.

Real Chrome checks retain bootstrap, binary/text transfer in both directions,
and add eight 4 MiB transfers from one page through fresh authenticated H3
connections. Every transfer checks its full length and SHA-256, as well as saved
methods after close. The additional test does not replace the existing bidirectional
checks.

## Local results

Linux amd64 / Go 1.27.1: bridge regressions, six JavaScript failure-path checks,
all six Chrome cases, full ordinary application suite, full serialized race suite
and go vet passed. One initial ordinary run failed before starting the local DERP
subprocess with `text file busy`. Its log is retained. Ten focused repetitions and
a subsequent full ordinary run passed without changing that test or native code;
the underlying intermittent process-start error is not claimed root-caused.

The first new bridge fixture also accidentally created an unobserved read Promise
in its pending-write case. Its rejected-Promise failure was retained; the fixture
now starts only the operation under test, with no production error suppression.

Local raw evidence is under `/tmp/tailcat-quic4-audit/`.

## Remote validation

Mac orchestrator: `audits/tailcat-quic4-20260927/` under the existing
`tailscale-all` checkout. Candidate builds use public immutable pins, record
the quic.3 base plus hashes of every changed source, and are explicitly test-only.
The remote worktree does not represent a tagged release package.

The host probe accepts an optional existing known-hosts file and applies strict
host-key verification to both SSH and SCP. This avoids changing the user's SSH
configuration or discarding verification when the current default file lacks
previously trusted host entries.

Mac native tests used the actual built macOS arm64 executable and passed, as did
all six real-Chrome cases and the six injected JavaScript failure paths.

### Direct UDP: all six runs passed

Three Linux amd64 hosts were used: Japan, Australia, and US. Each pair ran twice,
sequentially to avoid competing benchmarks.
Each iperf sample measured eight seconds after two omitted warm-up seconds.
Rates below are receiver Mbps; every sample is retained, including slower ones.

| Direction | Single stream, round 1 / 2 | Four streams, round 1 / 2 |
| --- | ---: | ---: |
| US → Australia | 277.80 / 296.20 | 354.07 / 349.28 |
| Australia → US | 254.30 / 272.63 | 284.55 / 204.68 |
| Australia → Japan | 238.30 / 249.67 | 250.79 / 248.93 |
| Japan → Australia | 375.93 / 368.29 | 391.23 / 334.38 |
| US → Japan | 239.87 / 279.50 | 247.22 / 290.40 |
| Japan → US | 334.21 / 306.55 | 288.19 / 319.73 |

Across these runs: 254,810,112 bytes passed SHA-256/length verification, including
both directions, concurrent transfers and recovery after idle. UDP delivered
360/360 probes (64/512/1200-byte payloads); loaded echo probes reported no errors.
Every run observed direct UDP and the native H3 engine, with no panic or cleanup
error. Loaded latency still varies: sample p95 reached 364.17 ms. This is not a
paired old/new comparison, a latency guarantee or proof of universal speedup.

### Forced relay: initial failure retained

The first AU/US forced-DERP run verified 16 MiB in each direction, then measured
23.18 Mbps US→AU. Its next iperf sample failed with `the server is busy running a
test. try again later`. The run is recorded as failed, and its missing reverse
throughput/UDP checks are not counted as passes. It reported no panic or cleanup
error. Both endpoint logs confirmed forced relay and no observed direct path.

The probe originally reused a single iperf listener for every sample. Its client
can finish before remote control-connection cleanup arrives through a relay. The
follow-up probe gives each sample its own listener, preserving original test
durations and the same application executable. This isolates measurements; it
does not establish the exact cause of the earlier cleanup timing.

The follow-up passed: US→AU 22.69 Mbps, AU→US 23.09 Mbps, 35,128,320 bytes
verified by length/SHA-256, concurrent and idle-recovery transfers passed, and
60/60 UDP probes arrived. Both endpoint logs confirmed forced relay and no
observed direct path. Loaded echo probes reported no errors (p95 up to 522.72 ms).
No panic or cleanup error occurred. The probe revision has SHA-256
`18411035ac035b0046a84ea965b357ceb87426da8fe96a2823883fad9023e1f1`.

The seven complete passing runs verified 289,938,432 bytes (about 276.5 MiB)
and delivered 420/420 UDP probes. The failed original relay run remains separate;
it is not relabeled as passed. All individual samples and artifact/source hashes
are in [the validation record](validation/browser-lifecycle-20260927.json).

Post-run host inventory found no current-run test directories or executables
still running. Five older test directories dated September 10/23 were identified
as pre-existing evidence and left intact.

The Singapore SSH preflight timed out. Bare SSH aliases were absent from the
Mac configuration; using the existing trusted host-key file allowed the three
hosts above to be checked. Singapore is not counted as validated.

### Exact candidate artifacts

- Linux amd64 SHA-256: `c8784ae1a95d1cec0cfb81bfc9ccafcc160ee6ca59553fdfa4fdd866b9559320`.
- macOS arm64 SHA-256: `5ba0999d2b31b6075331f2c21eec1a17c40c3c1ea3223e43a74025e0be35d1da`.
- UDP probe SHA-256: `6fba768f872954841f857d3781787e0b0713c3bcb8eb5faaf2d8c2ffe8635d78`.
- QUIC: `github.com/LiuTangLei/quic-go v0.63.0-quic.3`, revision `3d9e7d1f348241b89ec1060f733ea39b967e7f82`.
- Integration: `github.com/LiuTangLei/tailscale v1.102.5-0.20260923003659-b4f1aa3cd300`.
- Primitives: `github.com/LiuTangLei/wireguard-go v0.0.33-0.20260910045057-ed22747d204e`.

The candidate identifies itself as `v0.7.0-quic.4`, but `build.json` explicitly
records `test_only=true`, `source_dirty=true`, and no release archives. The code
identity is the quic.3 base plus the source-file hashes in `state.json`. This is
not an immutable application tag, public release or final package acceptance.
Windows/Android native execution, Safari/Firefox and multi-hour endurance were
not performed in this run. Publication remains a separate step after final
source/package verification.

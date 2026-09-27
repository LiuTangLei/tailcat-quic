# Tailcat-QUIC v0.7.0-quic.4 — validation scope

## Source and dependencies

This candidate fixes browser Promise and connection/listener callback retention,
and closes browser send connections on failure. Native transport, mandatory
connection secret/node authentication, reliable TCP streams, BBRv3 and FIN/ACK
behavior are unchanged. The clean source revision and all artifact hashes are
recorded in the accompanying `build.json`.

- QUIC: `github.com/LiuTangLei/quic-go v0.63.0-quic.3`, revision `3d9e7d1f348241b89ec1060f733ea39b967e7f82`.
- Integration: `github.com/LiuTangLei/tailscale v1.102.5-0.20260923003659-b4f1aa3cd300`.
- Primitives: `github.com/LiuTangLei/wireguard-go v0.0.33-0.20260910045057-ed22747d204e`.

Packages must use these public immutable module pins without local replacements.

## Completed runtime checks

`docs/browser-lifecycle-fix-20260927.md` and its machine-readable evidence record
the exact test candidate source hashes, retained failures and successful reruns:

- Linux ordinary/race/vet, actual WASM bridge lifecycle/memory regressions, six
  JavaScript failure paths, and six real Chrome cases passed.
- macOS arm64 native CLI/full tests, Chrome and JavaScript checks passed.
- Six direct runs on Japan/Australia/US and one forced-relay follow-up passed
  both-direction length/SHA-256, concurrent transfer, idle recovery and UDP checks.
  These verified 289,938,432 bytes and 420/420 UDP probes.
- The initial relay probe failure and initial local `text file busy` failure are
  retained. The former was followed by isolated iperf listeners; the latter did
  not recur in ten focused repetitions and a full rerun, and is not claimed
  root-caused.

## Package and throughput acceptance

Final acceptance must verify all seven executable archives and six Linux packages,
their hashes, immutable build metadata, and CLI tests using an executable extracted
from the actual archive. Cross-compilation alone is not native platform testing.
The separate acceptance report records these outcomes and the paired comparison
against official Tailcat v0.7.0 using userspace WireGuard. It must preserve both
directions, single/four streams, repetitions, slower samples and any failures.
The existing candidate WAN results are not a claim of sustained 500 Mbps.

Windows/Android device execution, Safari/Firefox and multi-hour endurance are
outside the completed scope. Publication is a separate explicit step; neither a
clean build nor a successful benchmark publishes a release or changes installers.

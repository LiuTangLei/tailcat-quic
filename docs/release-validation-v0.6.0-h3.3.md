# Tailcat H3 v0.6.0-h3.3 — release validation

Date: 2026-09-10. This release is based on official tailcat v0.6.0. It is an independent H3-only fork, not a Tailscale-supported product or a claim of complete WireGuard/VPN performance parity.

## Immutable source identity

| Component | Published pin | Source commit |
| --- | --- | --- |
| Official tailcat baseline | v0.6.0 | 790406204c002a6f109ef7c6a30a436601a1f6a8 |
| Reviewed Tailcat experiment | test/shared-h3-20260910 | ad864d56380ff43ecaa25ccb55322c5598374f13 |
| Shared QUIC, BBRv3 and passive batching | LiuTangLei/quic-go v0.62.0-tailcat.3 | cd7e74d71080a77e24e5786b8c6a66561c38a785 |
| Shared Tailscale transport plus release close fix | LiuTangLei/tailscale v1.102.3-tailcat.3 | 93ca289997691e28ca779947b08716888babe2df |
| Shared transport experiment before the close fix | shared/h3-performance-20260910 | 170d1bb356838d916154d5d8e6273642b89d4f05 |

The Tailcat release tag identifies the exact final application source. Its go.mod uses versioned public module replacements; it does not require a local go.work, sibling checkout, source overlay, or a filesystem replacement. GOWORK=off was used for release verification.

The QUIC pin includes the explicit withdrawal of active packet formation. In particular, this release does not restore connection_batch.go or the sendPacketsWithPortableBatch path that regressed reverse-direction throughput. It only combines already-ready queued packets, preserving packet boundaries, congestion/pacing decisions and receive scheduling. The separate, unvalidated read-buffer-pool staging branch is not part of this release.

## Product behavior and boundaries

TCP proxy connections use independent authenticated HTTP/3 CONNECT reliable byte streams on the QUIC connection. UDP/IP remains on CONNECT-IP / QUIC DATAGRAM. Both endpoints select the existing userspace BBRv3 controller. No AWG/WG data-plane fallback or transport selector is added.

The same mandatory connection credential, TLS-bound node proof, live node authorization and served-target checks apply. Stream buffering copies caller data before returning from Write. Half-close, read/write deadlines, final-data drain and peer revocation remain required behaviors. Numeric CLI `serve PORT` exposes TCP; the library's UDP API is tested separately and is not silently advertised as a numeric CLI UDP listener.

Reliable streams retain their existing long-connection policy and QUIC key updates; they do not inherit the full TLS-connection replacement timer intended for the standalone IP tunnel. The Tailscale general-purpose IP tunnel and its congestion-control defaults are outside this release's acceptance scope.

## Pre-existing shared WAN evidence

The shared experiment used two Linux hosts, anonymized here as client A and server B. Each timed test ran for 12 seconds after two seconds omitted startup, with loaded echo checks. Two rounds reversed baseline/candidate order. These measurements come from the completed shared experiment, not a new throughput benchmark run during packaging.

| Direction / streams | Previous H3 baseline, two runs (Mbps) | Final shared candidate, two runs (Mbps) |
| --- | ---: | ---: |
| A → B, 1 | 316.82 / 307.16 | 403.60 / 367.89 |
| A → B, 4 | 245.83 / 358.99 | 415.96 / 353.03 |
| B → A, 1 | 280.75 / 318.44 | 253.48 / 257.06 |
| B → A, 4 | 261.19 / 266.06 | 269.93 / 245.72 |

The mean forward single-stream gain was approximately 24%, while reverse single-stream throughput decreased approximately 15%. Server sampled peak RSS fell from 104.4 / 98.9 MiB to 79.6 / 71.6 MiB. Loaded p95 ranged from 171 to 351 ms and did not improve in every direction. No universal speedup, loss-free network, or production-scale fairness claim follows from these short runs.

Each run verified approximately 24.5 MiB of actual application content, covering bidirectional transfer, concurrent transfers and idle recovery; all SHA-256 checks passed. Forced DERP TCP measured 23.03 / 23.70 Mbps, and the separate UDP API returned all 60 echoes across three payload sizes. These are finite functional samples, not exhaustive network certification.

The active-batch experiment that cut reverse throughput to approximately 160–163 Mbps was rejected, not averaged into the successful candidate or quietly shipped. The shared source report is `LiuTangLei/tailscale/docs/h3-shared-performance-20260910.md` at commit 170d1bb356838d916154d5d8e6273642b89d4f05.

## Release-blocking defect found and fixed

The earlier unpublished h3.2 workflow failed a macOS race-test timeout and hung in Windows CLI TestPing. Review identified a deterministic deadlock in magicsock Bind.Close: it sent a wakeup into the one-entry DERP receive queue while holding the mutex, even when the queue was already full and the carrier reader had stopped. A receiver also needs that mutex to observe the closed state.

The new regression reproduced this defect before the fix. Close now sends the wakeup non-blockingly: an already queued item is itself sufficient, because receiveDERP checks the closed flag before processing any item. This preserves the close state and notification without requiring a reader to drain a full queue. The regression subsequently passed 50 race-instrumented repetitions. Full magicsock race tests and the relevant engine, QUIC-IP and H3 transport race tests also passed locally.

The small-round-trip fixture still verifies 60 sequential exchanges, now with distinct sequence contents, a five-second per-exchange deadline and a 90-second total context. The former shared 25-second deadline conflated cumulative runner scheduling with a per-operation progress failure. No performance acceptance threshold was silently claimed from this test. CLI ping subprocesses are now bounded independently so a future shutdown regression fails with captured output instead of wedging the full suite for 15 minutes.

## Local release verification

After switching to the public dependency pins, the following passed:

```sh
GOWORK=off go mod tidy
GOWORK=off go mod verify
GOWORK=off go test -count=1 -timeout=180s ./...
GOWORK=off go vet ./...
GOWORK=off go test -race -p=1 -parallel=1 -count=1 -timeout=240s ./...
```

The transport release also passed race tests for magicsock, wgengine, quicip and quicbind, including the new shutdown regression. Earlier shared dependency verification is recorded separately in the shared experiment report; this document does not relabel that work as a new test run.

## Mandatory publication gates

The Release workflow must pass native Linux/macOS/Windows ordinary tests, vet, race tests, module verification/tidiness and all seven cross-builds before GoReleaser creates a draft.

The draft is then downloaded independently on Linux, macOS and Windows. `scripts/verify-release-assets.py` verifies every asset checksum, the seven executable archives, the six Linux package files, required documentation/licenses, the native executable's exact version, clean VCS identity and both dependency pins. It then runs the CLI end-to-end suite against the **extracted executable**, using TAILCAT_TEST_BINARY; it does not substitute a freshly built program. These jobs are publication gates, and their final results are recorded on GitHub and in the release notes after packaging.

Targets are Linux amd64/arm64/armv7, macOS amd64/arm64, and Windows amd64/arm64. Native runner checks do not imply runtime coverage of every architecture. Browser/WASM, Nix packaging, Docker distribution and mobile application binaries are not supplied by this release. Executables are not claimed to be Apple-notarized or Authenticode-signed.

The h3.1 and h3.2 tags remain historical unpublished candidates; they are not evidence of successful earlier public releases. Public release status depends on the completed workflow and published GitHub Release, not the existence of a tag alone.

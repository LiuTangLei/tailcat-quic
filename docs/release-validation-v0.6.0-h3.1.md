# Tailcat H3 v0.6.0-h3.1 validation

Validation date: **2026-09-09**. This report records finite tests, not a claim that the implementation is defect-free.

## Source and dependency identity

| Component | Fixed source |
| --- | --- |
| Upstream tailcat baseline | `v0.6.0`, `790406204c002a6f109ef7c6a30a436601a1f6a8` |
| H3 transport library | `LiuTangLei/tailscale v1.102.3-tailcat.1`, `201834b1d02ad29d63b68c471a79d406d7647d22` |
| QUIC/BBRv3 library | `LiuTangLei/quic-go v0.62.0-tailcat.1`, `51f5f5143320456d4c90a524049819797c6e1523` |
| Tailcat fork release | The immutable `v0.6.0-h3.1` tag identifies the source packaged by the Release workflow. |

The release `go.mod` uses published module replacements, not filesystem paths or overlays. `go mod tidy`, `go mod verify`, and the complete tailcat suite passed after switching from development worktrees to these public dependencies.

## Functional and security checks

The full tailcat v0.6 test suite passes with the H3 integration. It exercises TCP/UDP service forwarding, direct-path discovery, ping, local forwards, SOCKS, file/SFTP operations, SSH authentication and terminal behavior, CLI arguments, saved keys, allowlists, shutdown, and datagram idle handling.

Additional tests reject legacy/unsupported connection-code versions, duplicate CBOR fields, malformed sizes, missing or zero secrets, modified bootstrap messages, reflected responses, and wrong server/client identities. H3 authentication also binds the application secret into the current TLS/node proof; successful discovery alone is not admission. Source-address and revocation tests are exercised in the transport dependency.

`FuzzH3ConnectionCode` completed **92,983 executions** in the recorded bounded fuzz run, with no crash or accepted-invalid-credential invariant failure. This is not exhaustive coverage of every byte string.

Disconnected and abandoned client identities can be reclaimed under capacity pressure after a grace period. A regression cycles through more than two full capacity windows while retaining an established session. Recent handshakes and established sessions are protected, retained identities remain bounded, and callback tests exercise concurrent updates without taking the policy mutex. Retirement atomically rechecks/reserves the lease, so a session established after a candidate scan cannot be erased by stale cleanup. The new establishment-versus-retirement regression passed 10,000 races across five instrumented test runs.

## Race and static checks

Tailcat passed:

```sh
go test -count=1 -timeout=180s ./...
go test -race -p=1 -parallel=1 -count=1 -timeout=240s ./...
go vet ./...
```

Race instrumentation serializes heavyweight operating-system/network fixtures; explicit stream/callback concurrency remains inside the tests. An inherited 50 ms UDP forwarding idle fixture was increased to 500 ms to leave scheduling headroom while retaining its five-second cleanup assertion. The first cold UDP exchange is allowed 30 seconds for H3 authentication, independently of the already completed discovery ping; subsequent data exchanges retain a five-second deadline. This separates cryptographic startup from steady-state forwarding instead of treating a discovery response as an established tunnel. SSH terminal execution was additionally repeated eight times under race instrumentation.

The H3 dependency passed race tests for `wgengine`, `wgtransport`, `wgtransport/nodeauth`, `wgtransport/quicbind`, `quicip`, `net/tsdial`, and `tailcfg`, plus relevant `go vet` checks. This includes wrong application credentials, actual live `bbr-v3` session statistics, replay/reflection/KCI, peer revocation, source-IP policy, session lifecycle, and rebind cases.

The QUIC dependency passed its full ordinary suite and `go vet ./...`. Its full race suite passes with **one allocation-measurement test**, `TestFrameParserAllocs`, excluded from race instrumentation and tested separately ten times normally. That test asserts zero allocations from a pool. [Go's race-enabled sync.Pool deliberately drops some pooled values](https://go.dev/src/sync/pool.go), so it is not a valid zero-allocation measurement under race. No functional QUIC test is excluded by this exception.

## BBRv3 coverage

The new controller has a separate v3 state machine, not a renamed BBRv1 sender. Deterministic tests cover STARTUP/DRAIN; DOWN/CRUISE/REFILL/UP; two-cycle bandwidth filtering; short- and long-term loss bounds; transmission-time loss thresholds; high-loss startup; upward-bound growth; ProbeRTT timing and half-BDP cwnd; idle restart; ACK aggregation; packet discard/duplicate handling; MTU changes; and saturating arithmetic.

The real-socket QUIC regression verifies **10 MiB per direction per scenario**, including four simultaneous streams and idle recovery. It runs normally and with a deterministic packet drop every 97 datagrams on both endpoints. All SHA-256 checks pass; both live endpoints report `bbr-v3`, and the loss case verifies that losses were actually injected.

The algorithm targets the BBRv3 revision-06 draft. This userspace QUIC adaptation is not Google's Linux TCP module. ECN response is not implemented/enabled for this controller, and transport-wide spurious-recovery undo is deliberately not inferred from a single late ACK. See the dependency's [BBRv3 implementation notes](https://github.com/LiuTangLei/quic-go/blob/v0.62.0-tailcat.1/BBRv3.md) for mapping and boundaries. These tests do not establish production-scale congestion fairness or superiority on every link.

## Cross-WAN application-data tests

The pre-publication development executables used the release build tags and `CGO_ENABLED=0`. A macOS arm64 client and a separate Linux amd64 host ran isolated temporary test processes. No production VPN process, routing table, firewall rule, or stored service configuration was changed. Test identities and private infrastructure are deliberately omitted.

| Path | Data integrity coverage | Verified application bytes |
| --- | --- | ---: |
| Direct UDP, H3-only on both endpoints | Two 32 MiB uploads, two 32 MiB downloads, four concurrent 2 MiB transfers, idle recovery in both directions, 1 KiB warm-up | 144,704,512 |
| Forced DERP, H3-only on both endpoints | One 4 MiB upload and download, four concurrent 2 MiB transfers, idle recovery in both directions, 1 KiB warm-up | 18,875,392 |

**All transferred data matched SHA-256.** Both direct-path endpoint logs confirmed the direct upgrade. Both relay-path endpoints had direct UDP disabled; data still passed over H3 through DERP. Neither path instantiated a WireGuard data-plane device. These are application-data checks, not merely discovery pings.

Observed single-transfer direct rates varied roughly from 11 to 21 Mbit/s in this particular run, and relay rates were lower and variable. They are not controlled throughput comparisons, service guarantees, or evidence that BBRv3 is faster than another controller. The primary acceptance criteria here were correct mode selection, bidirectional integrity, concurrency, idle recovery, and successful path operation.

## Packaging gates and platform scope

The Release workflow requires the Linux/macOS/Windows native test matrix, module portability/tidiness check, and all seven release-target cross-builds to pass before creating a draft. Executables target Linux amd64/arm64/armv7, macOS amd64/arm64, and Windows amd64/arm64. A cross-build alone is **not** a runtime test of that architecture.

Packaged-asset version, checksum, dependency-build-info, and final native smoke checks are recorded in the release notes after the draft is assembled. This source report does not pretend that an archive can contain the results of tests performed on that very archive after it was built.

Browser/WASM interoperability, every router/NAT/firewall topology, very long production uptime, and all target architectures on physical hardware were not exhaustively tested. Both peers must use this H3 fork and a supported `tch3…` code. The implementation does not add an AWG/WG data-plane fallback.

# tailcat-quic changelog

## v0.7.0-h3.2 (prerelease, 2026-09-22)

- Retain the complete upstream 0.7 merge and shared H3/QUIC optimizations below.
- Fix the final SSH shutdown race exposed by the h3.1 release gate: normal Close gives peer trailers/FIN a bounded receive drain instead of immediately resetting the peer's sender. Explicit cancellation remains an error; no delivery check is suppressed.
- Bound that drain by both five seconds and one MiB, using the existing reader and buffer. Verify full response/trailer delivery, silent-peer cleanup and continuing-peer budget exhaustion.
- Repeated the previously failing SSH forced-command test 40 times and SSH/exec race regressions three times; require new tagged and packaged platform gates before publication.

The final two-round h3.2 comparison improved US-to-AU means by about 6.3% (P1) and 4.0% (P4), but AU-to-US means declined about 11.9% and 25.4%. All complete samples and UDP/content checks are retained in the validation report. This release is explicitly a prerelease and does not replace the existing stable build.

The h3.1 candidate below was not published because its final macOS release gate failed. Its immutable source tag and failed workflow are retained. See [h3.2 validation](docs/release-validation-v0.7.0-h3.2.md).

## v0.7.0-h3.1 (unpublished candidate, 2026-09-22)

- Merge official Tailcat v0.7.0, including UDP exit-node forwarding, localhost/Windows fixes, older OpenSSH SFTP compatibility, `Server.Listen`, browse, exec and SSH forced-command support.
- Preserve the independent H3-only application: reliable TCP streams, QUIC DATAGRAM for UDP, TLS 1.3, node/connection-secret authentication and userspace BBRv3. Both endpoints use this fork; official `tc` codes remain incompatible.
- Pin shared QUIC 0.63 and the H3 integration library carrying the existing bounded I/O/lifecycle work. Backport matching upstream Android DNS/CA/netmon support and gVisor CUBIC/RACK clock fixes, without introducing duplicate dependency repositories.
- Fix H3 listener precedence and accepted-connection ownership; listener closure leaves accepted streams alive, while server shutdown releases them.
- Close temporary tunnel clients after the new DNS/SSH exposure probe, including rejected/timeout paths, preventing leftover DERP work.
- Remove forced legacy resolver build flags and keep release tags synchronized with the selected dependency's feature set. Verify every packaged native executable and its immutable dependency pins before publishing.

In the two-round AU/US comparison, US-to-AU four-stream mean increased from 328.16 to 354.15 Mbps, while single-stream mean decreased from 304.02 to 287.46 Mbps. Reverse means were nearly unchanged. Final runtime/UDP checks passed, but loaded latency still has spikes. This is not a universal speed-up guarantee. See [exact methods, samples and limitations](docs/release-validation-v0.7.0-h3.1.md).

## Earlier unreleased branding changes

- Rename the repository and public project name to `tailcat-quic`; the CLI remains `tailcat`.
- Rename the first public Release display title to `tailcat-quic v0.6.0`. Retain its immutable build tag, download assets and checksums.
- Simplify the README with quick usage, actual HTTP/3 traffic characteristics, TLS 1.3 encryption and session-bound node authentication.
- Update release tooling for the new repository and ordinary version numbers, without changing transport behavior or published dependencies.

## v0.6.0-h3.3 (2026-09-10)

First public release of the independent QUIC/HTTP/3-only fork, based on official tailcat v0.6.0. Both endpoints must use this H3 fork.

- Retain reliable per-connection H3 TCP streams, CONNECT-IP DATAGRAM for UDP/IP, and userspace BBRv3 on both endpoints. No AWG/WG fallback or additional transport configuration.
- Integrate the validated shared HTTP Datagram single-copy path, passive batching of already-ready packets, and reusable owned TCP write buffers. Do not include the rejected active packet-formation optimization or the unvalidated read-buffer-pool branch.
- Fix a reproduced shutdown deadlock when magicsock's DERP receive queue is already full. Preserve close notification while the underlying path is usable.
- Bound CLI test subprocesses and distinguish per-exchange progress from cumulative scheduling time in the small-round-trip regression.
- Pin public transport dependencies to `tailscale v1.102.3-tailcat.3` and `quic-go v0.62.0-tailcat.3`; builds require no local worktrees.
- Verify all release checksums and run CLI end-to-end tests against the actual packaged native executable on Linux, macOS and Windows before publication.

Performance is directional: the shared two-round experiment improved forward single-stream throughput by about 24% and reduced server peak RSS, but reverse single-stream throughput decreased about 15%. This is not a promise of universal improvement or general-purpose VPN parity. See [release validation](docs/release-validation-v0.6.0-h3.3.md).

The h3.1 and h3.2 tags below were development candidates whose release workflows did not complete; no public release was published for them.

## v0.6.0-h3.2 (unpublished candidate, 2026-09-10)

Performance release based on official tailcat v0.6.0; both endpoints should upgrade together.

- Carry TCP proxy connections on reliable HTTP/3 CONNECT streams in the authenticated QUIC session, avoiding a nested user-space TCP data path. UDP retains CONNECT-IP DATAGRAM semantics; BBRv3 remains the default on both endpoints.
- Preserve node/connection-secret authentication, per-service port restrictions, SSH peer identity and live revocation for the new TCP stream path. Public TLS connections cannot open an unauthenticated proxy.
- Implement bounded resumable-deadline queues, concurrent net.Conn contracts, TCP half-close, and real final-byte/FIN acknowledgment for one-shot shutdown.
- Keep long-lived SSH/file streams on their QUIC connection with packet-key updates; do not truncate them by periodically replacing the entire connection.
- Send connection-close notifications before closing magicsock, and release readers blocked behind full queues on peer shutdown.
- Fix incomplete-fragment capacity starvation while retaining bounded memory, and avoid unnecessary TCP fragmentation on the legacy IP path.
- Add official-WG versus H3 two-host benchmarks, role reversal, full-data hashes, UDP API and forced-DERP checks. The recorded candidate exceeded official WG throughput on the tested path; loaded latency still has spikes and is not guaranteed to match WG.
- Pin the published transport dependencies to `tailscale v1.102.3-tailcat.2` and `quic-go v0.62.0-tailcat.2`.

See [the v0.6.0-h3.2 validation report](docs/release-validation-v0.6.0-h3.2.md) for exact measurements, methodology and limitations.

## v0.6.0-h3.1 (unpublished candidate, 2026-09-09)

Initial independent H3-only candidate, based on official tailcat v0.6.0.

- Native-IP HTTP/3 CONNECT-IP and QUIC DATAGRAM replace the WireGuard data plane; no WG/AWG protocol fallback is started.
- Both QUIC endpoints explicitly select the new independent userspace BBRv3 controller. Published transport dependencies are fixed in go.mod; existing Tailscale and older BBR defaults remain unchanged.
- Versioned `tch3…` connection codes contain a mandatory secret credential. Bootstrap admission and the TLS-bound node proof both authenticate this secret; legacy official codes and disabled credentials fail closed.
- Reuse tailcat's direct-UDP discovery, DERP relay paths, TCP/UDP forwards, SOCKS, files/SFTP, SSH, saved identities, and node allowlists.
- Bound admission workers and retained identities, and reclaim disconnected identity history without evicting established sessions.
- Add Linux, macOS, and Windows executable packages, checksums, private-infrastructure-free validation notes, and publication gates that test before creating a draft release.

Both peers need this H3 fork. Standard WireGuard users should use official tailcat. See [the validation report](docs/release-validation-v0.6.0-h3.1.md) and [security notes](SECURITY.md) for test coverage and limitations.

The entries below describe the inherited upstream project before this fork.

## Unreleased

- `--serve=exit-node` servers now forward UDP flows; previously only
  TCP was forwarded, so DNS, QUIC, and other UDP traffic through an
  exit node went nowhere.
- Serving local ports works from Windows: the server resolves
  `localhost` itself instead of using the hosts file, which Windows
  ships without localhost entries, so the name no longer escapes to
  real DNS servers. It also dials both 127.0.0.1 and ::1, reaching
  services bound to only one loopback address. Official binaries also
  no longer build with the `netgo` tag that forced Go's pure resolver
  on Windows and macOS; they now use the operating system's resolver
  there, like a default `go build` does.
  ([#108](https://github.com/tailscale/tailcat/issues/108), reported
  by [@Sammy-T](https://github.com/Sammy-T))
- Go library: `Server.Status()` now includes a `Peer` entry per
  connected client, with `CurAddr` and `Relay` to tell a direct path
  from a DERP-relayed one.
  ([#116](https://github.com/tailscale/tailcat/issues/116), reported
  by [@Mo3he](https://github.com/Mo3he))
- Go library: the new `Server.Listen(ctx, network, address)` serves
  TCP and UDP ports in the standard `net.Listener` shape, as an
  alternative to the `OnTCP` and `OnUDP` hooks; for UDP, each Accept
  returns one client flow as a `net.Conn`. Listeners claim their
  specific ports ahead of the wildcard hooks, and Listen starts the
  server if it isn't running yet.
- `tailcat forward` takes an `--open-browser` flag that opens a web
  browser to the forwarded local port; `tailcat browse <tc-addr>` is
  an alias for `tailcat forward --open-browser <tc-addr> 0:80`.
- `exec` service: `tailcat serve exec -- <command>` runs the command
  for each incoming connection with the connection as its stdin and
  stdout, like inetd. With the `ssh` or `no-auth-ssh` service, the
  command after `--` instead replaces the shell for every session,
  like OpenSSH's `ForceCommand`, with no shell, client-chosen command,
  or SFTP offered.
- `tailcat ssh` to a DNS-named destination first probes the server the
  way a stranger would, with no credentials, and refuses to connect if
  the server hands out a shell to anyone, since an address published
  in DNS is public; `--skip-dns-safety-check` opts out. The README,
  the root help's DNS section, and `serve no-auth-ssh` startup now all
  warn that DNS-published addresses need `--allow` or
  `--ssh-authorized-keys`.
  ([#100](https://github.com/tailscale/tailcat/issues/100))
- Fixed argument parsing under Termux on Android, whose loader inserts
  the executable's path as an extra argument.
  ([#92](https://github.com/tailscale/tailcat/pull/92), [@shaunlee](https://github.com/shaunlee))
- The linux binaries now work when run directly on Android, under
  Termux, `adb shell`, or a rooted shell. Android has no
  `/etc/resolv.conf`, so a plain Go binary there could not resolve any
  name and failed at startup fetching the DERP map; it also found no
  CA roots and could not enumerate network interfaces. tailcat now
  links tailscale.com's `androiddns` and `androidbin` features, which
  detect Android at runtime, resolve names through Android's DNS
  resolver daemon, use the system certificate store, and fall back to
  a synthetic single interface. On regular Linux they do nothing.
  ([#117](https://github.com/tailscale/tailcat/issues/117), reported
  by [@risharde](https://github.com/risharde))
- Updated the tailscale.com dependency to its 2026-09-16 main branch,
  which brings data path performance work from wireguard-go and
  gVisor. wireguard-go now moves each batch of packets through one
  buffer of about 128 KiB instead of a separate buffer per packet,
  which in upstream's iperf3 benchmarks between two Linux machines
  raised throughput by 7% to 35% depending on the workload and cut
  peak memory for TCP transfers by half to three quarters. Small
  outbound packets such as keepalives and handshakes now use 2 KiB
  buffers, so packets waiting on a peer with no active session hold
  at least 97% less memory than before. gVisor's TCP stack, which
  carries every tailcat connection, now uses CUBIC congestion control
  and RACK loss detection; both had been switched off because of
  gVisor bugs that have since been fixed upstream.

## v0.6.0 (2026-09-04)

- Application-layer UDP support: servers can serve and forward UDP
  flows, clients can dial UDP ports, and `tailcat socks` supports
  SOCKS5 UDP ASSOCIATE. Idle incoming UDP flows close after a
  configurable timeout, two minutes by default. ([#25](https://github.com/tailscale/tailcat/pull/25), [@sksingh2005](https://github.com/sksingh2005))
- SSH public key authentication: `tailcat serve ssh` takes
  `--ssh-authorized-keys` with literal keys, key files, or
  `user@github` to fetch a GitHub user's keys. ([#88](https://github.com/tailscale/tailcat/pull/88))
- tailcat addresses now include a WireGuard pre-shared key by default;
  `--psk=false` opts out, and servers warn when serving without one. ([#85](https://github.com/tailscale/tailcat/pull/85))
- `tailcat forward` can forward through exit-node servers to arbitrary
  IP:port targets, and a local port of 0 picks a free one. ([#75](https://github.com/tailscale/tailcat/pull/75), [@Audi-dask](https://github.com/Audi-dask))
- `--derpmap-url` defaults from the `TAILCAT_DERPMAP_URL` environment
  variable. ([#72](https://github.com/tailscale/tailcat/pull/72))
- Served processes receive the authenticated peer's node key in
  `TAILCAT_PEER_KEY`, in the same `nodekey:...` form `--allow` takes.
  ([#89](https://github.com/tailscale/tailcat/pull/89), [@seffs](https://github.com/seffs))
- `tailcat genkey --embed-derp-map` no longer panics when no fixed
  region is set, and unknown `--region` values report an error naming
  the missing region. ([#91](https://github.com/tailscale/tailcat/pull/91), [@gmkbenjamin](https://github.com/gmkbenjamin); reported by [@chanchiwai-ray](https://github.com/chanchiwai-ray))
- Proxied TCP connections finish their teardown instead of losing data
  queued at close. ([#83](https://github.com/tailscale/tailcat/pull/83))
- Connection setup resends pings lost by busy relays instead of
  waiting out whole timeouts, and SOCKS dials get a longer budget than
  a single WireGuard handshake. ([#71](https://github.com/tailscale/tailcat/pull/71))
- Packaging: the Nix flake builds in CI with Go built from source and
  an automatically refreshed vendor hash; conda-forge installation is
  documented ([#76](https://github.com/tailscale/tailcat/pull/76), [@pavelzw](https://github.com/pavelzw)); the test suite is hermetic and several times
  faster, for reliable distro package builds.

## v0.5.0 (2026-09-02)

- New `tailcat forward` subcommand: listen on a local TCP port and
  forward each connection to a tailcat server. ([#62](https://github.com/tailscale/tailcat/pull/62), [@Audi-dask](https://github.com/Audi-dask))
- The connection string is now called a "tailcat address" everywhere;
  the old flag spellings remain as hidden aliases.
- Write-only file shares are actually write-only: drop boxes no longer
  leak whether a file already exists, and the new `:wo+` mode allows
  overwrites.
- Mistyped addresses no longer fall through to DNS lookups.
- Hardened validation of arguments passed to ssh and scp child
  processes.
- Added SECURITY.md.
- Windows: the SSH server looks for pwsh.exe. ([#61](https://github.com/tailscale/tailcat/pull/61), [@gcurtis](https://github.com/gcurtis))

## v0.4.0 (2026-08-31)

- File transfer: servers share a directory with
  `tailcat serve --files=DIR` (read-only, read-write, or write-only
  drop box modes) over the SSH SFTP subsystem, and new `cp`, `recv`,
  and `ls` subcommands use it. ([#48](https://github.com/tailscale/tailcat/pull/48))
- New `serve` subcommand as the long-form way to run servers.
- The CLI was rewritten with declarative subcommands: fuller help with
  examples on every subcommand, and requested help prints to stdout.
- SSH support on Windows, and CI now tests macOS and Windows; the
  integration tests were made Windows-portable. ([#34](https://github.com/tailscale/tailcat/pull/34), [@FenjuFu](https://github.com/FenjuFu))
- Official release binaries build with a trimmed feature set for
  smaller size; the tag list is documented in build-tags.md.
- Addresses with null DERP regions or nodes are rejected, and the meow
  packet encoding gained tests. ([#52](https://github.com/tailscale/tailcat/pull/52), [#44](https://github.com/tailscale/tailcat/pull/44), [@CharmingGroot](https://github.com/CharmingGroot))
- `genkey --client --key=default` is rejected as a likely mix-up, and
  `--version` keeps working as an unadvertised alias.
- The Nix flake's vendor hash stays fresh automatically.
- Documented Homebrew installation for macOS.

## v0.3.0 (2026-08-30)

- Node and disco keys are now separate, matching Tailscale's split
  between identity and path discovery.
- Local dev DERP mode is usable end to end and tested hermetically.
- Clients drain the final ACK before exiting, so the last bytes of a
  transfer are not lost at close.
- Documented the Arch Linux package. ([#30](https://github.com/tailscale/tailcat/pull/30), [@BarbUk](https://github.com/BarbUk))

## v0.2.0 (2026-08-30)

- Addresses with malformed public keys are rejected with a clear
  error instead of failing later. ([#26](https://github.com/tailscale/tailcat/pull/26), [@keyurbodar](https://github.com/keyurbodar))
- Fixed a close panic in the browser (Wasm) build.
- Documented the prebuilt binaries and container image.

## v0.1.0 (2026-08-30)

First tagged release. Highlights:

- Release process: static Linux binaries, deb and rpm packages,
  Windows zips, checksums, and container images on ghcr.io.
- Browser demo published to GitHub Pages, with the demo reusable by
  other servers.
- A Nix flake.
- DERP maps are cached on disk with ETag revalidation, plus a
  process-wide in-memory cache.
- `ping` reports the network path and gained `--until-direct`; peers
  advertise their endpoints on important events so direct paths form
  reliably.
- `genkey --fixed-region` bakes a region choice into a key.
- The Go library's zero value `Server` and `Client` are usable
  directly.
- Closing a `Server` closes its active connections and backend
  resources. ([#19](https://github.com/tailscale/tailcat/pull/19), [@0xcadams](https://github.com/0xcadams))
- SOCKS mode runs without a child command, takes a custom listen
  address ([#20](https://github.com/tailscale/tailcat/pull/20), [@tw4452852](https://github.com/tw4452852)), and uses the configured client key.
- The ssh ProxyCommand forwards a custom DERP map ([#22](https://github.com/tailscale/tailcat/pull/22), [@zukka77](https://github.com/zukka77)) and uses
  a short deterministic ControlPath ([#15](https://github.com/tailscale/tailcat/pull/15), [@korjavin](https://github.com/korjavin)).
- README fixes. ([#21](https://github.com/tailscale/tailcat/pull/21), [@CooperSheroy](https://github.com/CooperSheroy))

## derpcat becomes its own Go module (2026-03-07)

- Left the tailscale.com fork: commit
  [68ac83e73](https://github.com/tailscale/tailcat/commit/68ac83e73)
  ("derpcat: use tailscale.com as a library instead of forking") made
  derpcat its own Go module, depending on tailscale.com as a regular
  library. Renamed to tailcat in
  [e6b242f14](https://github.com/tailscale/tailcat/commit/e6b242f14)
  (2026-07-17). ([@bradfitz](https://github.com/bradfitz))

## derpcat (2023-09-14)

- First worked, as "derpcat" inside a fork of the tailscale.com repo,
  in commit
  [911915fbb](https://github.com/tailscale/tailcat/commit/911915fbb)
  ("derpcat: it's alive!"), written on UA 605 PDX-ORD without buying
  the wifi. ([@bradfitz](https://github.com/bradfitz))

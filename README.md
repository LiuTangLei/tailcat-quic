# tailcat-quic

**English** · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

An independent QUIC/HTTP/3-only fork of [Tailscale Tailcat v0.7.0](https://github.com/tailscale/tailcat/tree/v0.7.0).

Tailcat-QUIC keeps Tailcat's account-free, control-plane-free peer-to-peer model, but replaces the WireGuard data plane with authenticated HTTP/3 and QUIC. TCP services use reliable HTTP/3 streams, UDP/IP uses QUIC DATAGRAM, TLS 1.3 protects the QUIC session, and userspace BBRv3 is the default congestion controller.

**Current release: [v0.7.0-quic.2](https://github.com/LiuTangLei/tailcat-quic/releases/tag/v0.7.0-quic.2)**

> Both peers must run this fork. `tch3…` addresses are intentionally incompatible with upstream Tailcat's WireGuard `tc…` addresses.

## Why this fork

- Real QUIC + HTTP/3 traffic instead of WG/AWG on the data plane.
- TLS 1.3 plus connection-secret and node authentication.
- Reliable HTTP/3 streams for TCP services; QUIC DATAGRAM for UDP.
- Userspace BBRv3.
- Direct UDP NAT traversal when possible, DERP relay fallback when needed.
- Upstream Tailcat 0.7 features: TCP/UDP listeners, exit-node UDP forwarding, browse, exec/forced-command SSH, SFTP compatibility, Windows localhost fixes, Android/Termux runtime helpers, and peer path status.
- Bounded stream-buffer reuse and already-ready read coalescing to reduce allocations and small hand-offs without adding a batching timer.

## Install

### Fastest install

Linux and macOS:

~~~sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
~~~

Windows PowerShell:

~~~powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
~~~

Both installers download the selected GitHub Release asset, verify its SHA-256 against `checksums.txt`, verify `tailcat version`, and do **not** start a service or change firewall/routing settings.

### Package managers and one-command installs

| Method | Command | Status |
| --- | --- | --- |
| Release archives | [GitHub Releases](https://github.com/LiuTangLei/tailcat-quic/releases) | Linux amd64/arm64/armv7, macOS amd64/arm64, Windows amd64/arm64 |
| Debian / Ubuntu | download the matching `.deb` from Releases | amd64/arm64/armv7 |
| Fedora / RHEL | download the matching `.rpm` from Releases | amd64/arm64/armv7 |
| Homebrew | `brew tap LiuTangLei/tailcat-quic https://github.com/LiuTangLei/tailcat-quic.git && brew install LiuTangLei/tailcat-quic/tailcat-quic` | in-repo tap formula |
| Scoop | `scoop install https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/bucket/tailcat-quic.json` | direct manifest |
| Nix | `nix profile install github:LiuTangLei/tailcat-quic/quic-v0.7#tailcat-quic` | repository flake |
| Go toolchain | `go run github.com/LiuTangLei/tailcat-quic/install@v0.7.0-quic.2` | verified source bootstrap |
| Container | `docker run --rm -i ghcr.io/liutanglei/tailcat-quic:latest` | Linux amd64/arm64 GHCR image |

Do **not** add `-t` to the container command when piping tunnel data: a PTY merges stderr status output with stdout tunnel data.

The upstream documentation also lists Snap, AUR and conda-forge. Those are external registries with their own account/review process; this README does not claim those commands until the QUIC package is actually published there. See [INSTALL.md](INSTALL.md) for the exact parity matrix and maintainer publication requirements.

### Platform coverage

The downloadable release covers every upstream prebuilt platform and adds macOS archives:

- Linux: amd64, arm64, armv7 — tar.gz, deb and rpm.
- Windows: amd64, arm64 — zip.
- macOS: amd64, arm64 — tar.gz.

The CLI also cross-builds for FreeBSD/OpenBSD amd64 and arm64. The browser/WebAssembly bundle builds from source; browser runtime interoperability is tracked separately because browsers are relay-only and require a real browser integration test.

## Quick start

Serve a local port:

~~~sh
tailcat serve 8080
~~~

Share the complete `tch3…` address with the other peer over a trusted channel, then forward it locally:

~~~sh
tailcat forward 'tch3…' 18080:8080
# open http://127.0.0.1:18080
~~~

Open a remote HTTP service directly in your browser:

~~~sh
tailcat browse 'tch3…'
~~~

Serve SSH with explicit authorization:

~~~sh
tailcat serve --ssh-authorized-keys ~/.ssh/authorized_keys ssh
tailcat ssh 'tch3…'
~~~

Run a fixed command per incoming connection:

~~~sh
tailcat serve exec -- /path/to/program arg1 arg2
~~~

Use `tailcat --help` and each subcommand's `--help` for file transfer, SOCKS5, exit-node forwarding and advanced authorization options.

## Performance

The `v0.7.0-quic.2` read-path optimization reuses bounded 32 KiB stream buffers and coalesces only data that is already ready. It does not delay reads to form larger batches.

In the release A/B test between AU and US1420, four-stream means improved from **333.41 → 368.47 Mbps** in one direction and **211.56 → 305.86 Mbps** in the other. Single-stream means were **299.36 → 299.41 Mbps** and **231.34 → 262.86 Mbps**. These are limited WAN samples, not a universal speed guarantee.

The final public-pin Linux release binary was separately measured at **327.43 / 321.95 Mbps** with four TCP streams in the two directions. See [the full validation report](docs/release-validation-v0.7.0-quic.2.md) for methodology, slower samples, loaded latency and limitations.

## Security model

HTTP/3 is genuine protocol framing, not a promise of being indistinguishable from every browser. Private hostnames, ports, packet sizes/timing, discovery traffic and relay behavior can still be observable.

- TLS verification, connection-secret binding and node authorization remain enabled.
- `--psk=false` is rejected in this fork.
- TCP streams are opened only on an already authenticated QUIC session and are checked against the live service/peer policy.
- UDP/IP remains authenticated CONNECT-IP / QUIC DATAGRAM traffic.
- No WireGuard/AWG data-plane fallback is enabled.
- A `wireguard-go` dependency may still provide TUN/network primitives; that does **not** add a second WireGuard encryption layer.

Treat a `tch3…` connection address as a credential. See [SECURITY.md](SECURITY.md).

## Build from source

Go 1.27.1 is required.

~~~sh
git clone https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
git checkout quic-v0.7
go build -trimpath -tags "$(cat build-tags.txt)" -o tailcat ./cmd/tailcat
./tailcat version
~~~

Release builds use immutable public dependency pins. The application lives only in this repository; the shared protocol implementation is maintained in `LiuTangLei/quic-go` and the H3 integration in `LiuTangLei/tailscale`.

## Release and packaging

The release assets are locally validated before publication. A minimal public-repository GitHub workflow may be used for bounded platform/container jobs on standard GitHub-hosted runners; larger/GPU runners and heavy per-commit builds are intentionally avoided. The shared `quic-go` repository remains automation-free.

See [INSTALL.md](INSTALL.md) for installation details and [docs/manual-release.md](docs/manual-release.md) for maintainer release steps.

This is an independent fork and is not supported by Tailscale. Original copyright and BSD-3-Clause terms are in [LICENSE](LICENSE); dependency notices are in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

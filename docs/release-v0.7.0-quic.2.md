# tailcat-quic v0.7.0-quic.2

[English README](../README.md) · [简体中文](../README.zh-CN.md) · [日本語](../README.ja.md)

This is the first Tailcat 0.7 release using the requested `quic.N` naming. It is based on upstream Tailcat v0.7.0 and keeps the independent QUIC/HTTP/3-only data plane: reliable authenticated HTTP/3 streams for TCP, QUIC DATAGRAM for UDP/IP, TLS 1.3, connection-secret + node authentication, and userspace BBRv3.

Both peers must run this fork. Existing `tch3…` addresses keep the same protocol prefix; they are not compatible with upstream WireGuard `tc…` addresses.

## Highlights

- Merged upstream Tailcat 0.7 features, including TCP/UDP `Server.Listen`, UDP exit-node forwarding, Windows localhost fixes, older OpenSSH/SFTP compatibility, browse, exec/SSH forced commands, Android/Termux helpers, and peer path status.
- Fixed H3 listener ownership so accepted connections survive the callback that handed them to the listener.
- Fixed temporary client cleanup in DNS/SSH safety probes.
- Preserved bounded final-byte/FIN delivery for reliable H3 streams without treating resets as successful delivery.
- Added a bounded H3 read-buffer cache: at most 64 idle 32 KiB blocks (2 MiB per backend), with used bytes cleared before cross-stream reuse.
- A `Read` can now copy additional **already-ready** H3 chunks into the caller's buffer. It never waits to form a larger batch, so interactive latency is not traded for throughput.
- Shared QUIC is pinned to `LiuTangLei/quic-go v0.63.0-quic.2`.
- The application remains in `LiuTangLei/tailcat-quic`; no duplicate Tailcat application repository is required.

## Performance validation

AU was the server and US1420 the client. The old/new order was alternated (`old → new`, then `new → old`), with two complete runs per build. Each throughput sample used 10 measured seconds plus 2 explicitly omitted warm-up seconds.

| Data direction | Streams | Previous 0.7 candidate mean | quic.2 mean |
| --- | ---: | ---: | ---: |
| US1420 → AU | 1 | 299.36 Mbps | 299.41 Mbps |
| US1420 → AU | 4 | 333.41 Mbps | **368.47 Mbps** |
| AU → US1420 | 1 | 231.34 Mbps | **262.86 Mbps** |
| AU → US1420 | 4 | 211.56 Mbps | **305.86 Mbps** |

The final Linux release binary, built from public immutable module pins, was separately measured at **327.43 / 321.95 Mbps** with four TCP streams in the two directions. The direct UDP API test returned **60/60** echoes across 64, 512 and 1200-byte payloads.

These are limited WAN samples, not a guarantee for every network or a claim of zero loss/low tail latency. The release does not claim that the older SG↔US path variability is fixed. See [the full validation report](release-validation-v0.7.0-quic.2.md).

## Install

Linux / macOS:

~~~sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
~~~

Windows PowerShell:

~~~powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
~~~

Package-manager commands, Nix, Homebrew, Scoop, container usage and the exact upstream install-parity matrix are documented in [INSTALL.md](../INSTALL.md).

## Release assets

- Linux amd64 / arm64 / armv7: tar.gz, deb and rpm.
- Windows amd64 / arm64: zip.
- macOS amd64 / arm64: tar.gz.
- `checksums.txt` verifies all distributable packages.
- `build.json` records the exact source, toolchain, dependency pins and hashes.
- `VALIDATION.md` records native package and WAN verification.

The release therefore covers every upstream prebuilt platform and additionally ships macOS archives. FreeBSD/OpenBSD source cross-builds pass; the browser/WASM bundle builds from source and is checked separately with a real browser integration job.

## Security and compatibility

- `--psk=false` remains rejected.
- TCP streams are admitted only on an authenticated QUIC session and still pass live peer/service authorization.
- UDP/IP uses authenticated CONNECT-IP / QUIC DATAGRAM.
- No WG/AWG data-plane fallback is enabled.
- HTTP/3 is genuine protocol framing, but this project does not promise perfect indistinguishability from every browser.
- Treat the full `tch3…` address as a credential.

See [SECURITY.md](../SECURITY.md).

## Build / package automation

The default release gate remains real local/host validation. For install parity, the public application repository may use **bounded standard GitHub-hosted runners** for platform compatibility and the amd64/arm64 GHCR container image. Standard runners are free for public repositories under GitHub's current rules; larger/GPU runners are not used.

The GHCR workflow uses only the automatically generated `GITHUB_TOKEN` with `contents: read` and `packages: write`. No server SSH key, Tailcat credential or long-lived PAT is required. The shared `quic-go` repository remains automation-free.

Snap, AUR and conda-forge are external registries. Their commands are not advertised as live until those registries actually contain `tailcat-quic`; maintainer requirements are documented in [INSTALL.md](../INSTALL.md).

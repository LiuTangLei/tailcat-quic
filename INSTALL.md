# Installing tailcat-quic

This page documents the install methods for `LiuTangLei/tailcat-quic`. It is **not** the upstream WireGuard Tailcat package. Both peers must use this QUIC fork and its `tch3…` addresses.

Current release: **`v0.7.0-quic.2`**.

## Support target

The project tracks the installation/platform matrix documented by upstream Tailcat. The current downloadable release is not narrower than upstream's prebuilt matrix and additionally publishes macOS archives.

| Method / platform | Upstream Tailcat | tailcat-quic | Notes |
| --- | --- | --- | --- |
| Linux amd64 static | yes | yes | tar.gz |
| Linux arm64 static | yes | yes | tar.gz |
| Linux armv7 static | yes | yes | tar.gz |
| Linux deb/rpm, same arches | yes | yes | Release assets |
| Windows amd64 | yes | yes | zip |
| Windows arm64 | yes | yes | zip |
| macOS amd64 / arm64 archive | Homebrew upstream | yes | extra tar.gz assets |
| FreeBSD/OpenBSD source build | yes | cross-builds pass | amd64/arm64 checked |
| Browser js/wasm source build | yes | bundle builds | relay-only; runtime browser integration is tracked separately |
| Homebrew | yes | yes | in-repo tap formula |
| Scoop | yes | yes | direct manifest URL |
| Container | yes | yes | GHCR amd64/arm64 |
| Nix | yes | yes | repository flake |
| Go toolchain | yes | yes | verified bootstrap module |
The support goal is platform parity, not one-for-one duplication of every third-party package registry. Linux, macOS and Windows all have verified direct installers, while Release assets and the package-manager methods below provide additional choices.

## Fast verified installer

### Linux / macOS

~~~sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
~~~

Optional explicit version or install directory:

~~~sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh -o /tmp/tailcat-install.sh
sh /tmp/tailcat-install.sh --version v0.7.0-quic.2 --bin-dir "$HOME/.local/bin"
~~~

The installer downloads the release archive plus `checksums.txt`, verifies SHA-256 and the executable's reported version, then atomically installs `tailcat`. It does not start a service or change networking.

### Windows PowerShell

~~~powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
~~~

The default location is under the current user's `LocalAppData\Programs\tailcat-quic`. The installer validates SHA-256 and version, updates the user PATH unless disabled, and requires no administrator access.

## Prebuilt release packages

[GitHub Releases](https://github.com/LiuTangLei/tailcat-quic/releases) provides:

- Linux amd64 / arm64 / armv7: tar.gz, deb and rpm.
- Windows amd64 / arm64: zip.
- macOS amd64 / arm64: tar.gz.
- `checksums.txt` for all distributable packages.

Verify `tailcat version` after installation. For `v0.7.0-quic.2` it must print exactly:

~~~text
v0.7.0-quic.2
~~~

The Linux package name is `tailcat-quic` but the installed command is `tailcat`, so it conflicts with an installed upstream Tailcat package.

## Homebrew

The project carries a tap-compatible formula in `Formula/tailcat-quic.rb`.

One-line install:

~~~sh
brew tap LiuTangLei/tailcat-quic https://github.com/LiuTangLei/tailcat-quic.git &&   brew install LiuTangLei/tailcat-quic/tailcat-quic
~~~

The explicit URL is intentional: the application repository is named `tailcat-quic` rather than creating a duplicate `homebrew-tailcat-quic` source repository.

## Scoop

Scoop accepts a manifest URL directly:

~~~powershell
scoop install https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/bucket/tailcat-quic.json
~~~

The manifest selects amd64 or arm64 and verifies the release SHA-256.

## Container

~~~sh
docker pull ghcr.io/liutanglei/tailcat-quic:latest
docker run --rm -i ghcr.io/liutanglei/tailcat-quic:latest
~~~

A versioned tag is also published:

~~~sh
docker run --rm -i ghcr.io/liutanglei/tailcat-quic:v0.7.0-quic.2
~~~

Do not add `-t` when piping tunnel data. A PTY merges stderr status output with stdout tunnel data.

The container is published only for linux/amd64 and linux/arm64, matching upstream's container platform set.

## Nix

Run directly:

~~~sh
nix run github:LiuTangLei/tailcat-quic/quic-v0.7
~~~

Install into a profile:

~~~sh
nix profile install github:LiuTangLei/tailcat-quic/quic-v0.7#tailcat-quic
~~~

The flake selects verified release binaries for Linux amd64/arm64/armv7 and macOS amd64/arm64.

## Go toolchain / source-only operating systems

Direct `go install github.com/LiuTangLei/tailcat-quic/cmd/tailcat@...` is intentionally not used because the application pins forked public dependencies with Go `replace` directives.

Use the small verified bootstrap module instead:

~~~sh
go run github.com/LiuTangLei/tailcat-quic/install@v0.7.0-quic.2
~~~

It downloads the exact source module through the Go module system, checks the recorded module checksum, builds the real `cmd/tailcat` with the release tags, verifies its version and installs it to `GOBIN` / `GOPATH/bin`.

This is the source-build route for FreeBSD and OpenBSD as well. amd64 and arm64 cross-builds are checked before release.

## Browser / WebAssembly

The `web` package builds for `js/wasm`:

~~~sh
GOOS=js GOARCH=wasm go build ./web
~~~

Browser traffic is relay-only. A successful WASM build is not reported as a full runtime browser validation; the manual platform workflow runs the real headless-browser integration test separately.

## Android / Termux

Linux binaries retain upstream 0.7's runtime Android helpers for DNS, CA roots and restricted network-interface discovery. This is a command-line binary, not an Android VPN application.

## GitHub Actions and secrets

This is a public repository. Standard GitHub-hosted runners are free for public repositories under GitHub's current billing rules; larger/GPU runners and storage have separate rules.

The bounded GHCR workflow needs **no manually created repository secret**:

- `GITHUB_TOKEN` is created automatically for each workflow run.
- The job requests only `contents: read` and `packages: write`.
- No server SSH keys, Tailcat connection codes or long-lived GitHub PATs are stored.

The shared `LiuTangLei/quic-go` repository remains automation-free.

## Build from a clone

~~~sh
git clone https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
git checkout quic-v0.7
go build -trimpath -tags "$(cat build-tags.txt)" -o tailcat ./cmd/tailcat
./tailcat version
~~~

See [README.md](README.md), [SECURITY.md](SECURITY.md) and [docs/release-validation-v0.7.0-quic.2.md](docs/release-validation-v0.7.0-quic.2.md).

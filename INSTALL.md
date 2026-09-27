# Install Tailcat-QUIC

Install on both machines, then use `tailcat` as shown in the [quick start](README.md#quick-start). The current release is [v0.7.0-quic.4](https://github.com/LiuTangLei/tailcat-quic/releases/tag/v0.7.0-quic.4).

## Linux and macOS

```sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
```

The script selects your OS and architecture, checks SHA-256 and the executable version, and installs the command. Running it again updates the command and keeps your saved keys. It prefers a writable standard directory already on `PATH`, falling back to `~/.local/bin` for a regular user or `/usr/local/bin` for root. In Termux it uses `$PREFIX/bin`.

If the fallback directory is not on `PATH`, the installer prints the directory to add to your shell configuration. A custom destination or a pinned version is optional:

```sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh -o /tmp/tailcat-install.sh
sh /tmp/tailcat-install.sh --version v0.7.0-quic.4 --bin-dir "$HOME/.local/bin"
```

## Windows

Run in PowerShell:

```powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
```

It installs for the current user, verifies the download and adds the command to the user `PATH`. Open a new terminal if needed. Administrative privileges and execution-policy changes are unnecessary.

## Prebuilt packages

Download from [Releases](https://github.com/LiuTangLei/tailcat-quic/releases/latest):

| Platform | Architectures | Packages |
| --- | --- | --- |
| Linux | amd64, arm64, armv7 | tar.gz, DEB, RPM |
| macOS | Intel, Apple Silicon | tar.gz |
| Windows | amd64, arm64 | ZIP |

Archives contain the `tailcat` executable and documentation. For DEB/RPM, install the downloaded package with your usual package manager, for example `sudo apt install ./tailcat-quic_0.7.0-quic.4_amd64.deb` or `sudo dnf install ./tailcat-quic_0.7.0-quic.4_amd64.rpm`.

Both the official package and this fork provide a command named `tailcat`; choose which one your machines use. The packages do not start a background service.

## Homebrew

```sh
brew tap LiuTangLei/tailcat-quic https://github.com/LiuTangLei/tailcat-quic.git
brew install LiuTangLei/tailcat-quic/tailcat-quic
```

Update with `brew update && brew upgrade tailcat-quic`.

## Scoop

```powershell
scoop install https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/bucket/tailcat-quic.json
```

## Nix

```sh
nix run github:LiuTangLei/tailcat-quic/quic-v0.7
# Or install:
nix profile install github:LiuTangLei/tailcat-quic/quic-v0.7#tailcat-quic
```

## Container

```sh
docker run --rm -i ghcr.io/liutanglei/tailcat-quic:latest
```

Linux amd64 and arm64 images are available. Use `:v0.7.0-quic.4` to pin a version. Keep `-i` for stdin; omit `-t` when transferring data, because a terminal can mix status messages into the byte stream.

## Build from source

With Go 1.27.1 or newer:

```sh
go run github.com/LiuTangLei/tailcat-quic/install@v0.7.0-quic.4
```

This builds the pinned source and installs `tailcat` to `GOBIN` or `GOPATH/bin`. It also supports source builds on FreeBSD and OpenBSD. The small bootstrap handles this fork's dependency replacements; direct `go install …/cmd/tailcat@version` does not support them.

To build from a checkout instead:

```sh
git clone --branch v0.7.0-quic.4 --depth 1 https://github.com/LiuTangLei/tailcat-quic.git
cd tailcat-quic
go build -trimpath -tags "$(cat build-tags.txt)" -ldflags='-X main.version=v0.7.0-quic.4' -o tailcat ./cmd/tailcat
```

Android/Termux uses the CLI's Linux runtime helpers; it is not an Android VPN app. Browser/WASM is covered under [Experimental features](docs/experimental.md).

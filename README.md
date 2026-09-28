# Tailcat-QUIC

**English** · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

Connect two machines, forward a port, or send a file with a connection address. No account, VPN setup, or administrator access is needed to run it. Direct connections and relay fallback are automatic.

This independent fork of [Tailcat](https://github.com/tailscale/tailcat) uses authenticated QUIC/HTTP/3 with BBRv3. The command is still `tailcat`, with the same everyday workflow as upstream. **Both ends need Tailcat-QUIC**: its `tch3…` addresses are incompatible with upstream's `tc…` addresses.

[Latest release: v0.7.0-quic.4](https://github.com/LiuTangLei/tailcat-quic/releases/tag/v0.7.0-quic.4)

## Install

Linux / macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/LiuTangLei/tailcat-quic/quic-v0.7/install.ps1 | iex
```

The installers select your platform and verify the download. Archives, DEB/RPM, Homebrew, Scoop, Nix and containers are covered in [Installation](INSTALL.md).

## Quick start

### Send text or a file

On the receiving machine:

```sh
tailcat > received.txt
```

It prints a `tch3…` address. Copy the complete address to the sending machine:

```sh
echo hello | tailcat 'tch3…'
# Or send a file:
tailcat 'tch3…' < document.pdf
```

Run the receiver again for another transfer. Fresh keys are generated automatically. Share the address privately: it grants access to the service you started.

### Forward a local service

On the machine running your service:

```sh
tailcat serve 8080
```

On the other machine:

```sh
tailcat forward 'tch3…' 18080:8080
```

Open `http://127.0.0.1:18080`. For a web service, `tailcat browse 'tch3…'` can open it directly.

### Copy files by name

```sh
# Receiver
tailcat recv

# Sender
tailcat cp document.pdf 'tch3…:'
```

The receiver saves uploads in its current directory with unique names. To offer existing files for download, run `tailcat serve --files=./shared files`; the directory is read-only by default. The peer can use `tailcat ls 'tch3…:'` and `tailcat cp 'tch3…:document.pdf' .`.

### SSH

Use your existing authorized keys:

```sh
tailcat serve --ssh-authorized-keys ~/.ssh/authorized_keys ssh
```

Then connect with `tailcat ssh 'tch3…'`. Alternatively, expose an existing SSH server with `tailcat serve 22`.

## More commands

| Task | Command |
| --- | --- |
| Check the connection path | `tailcat ping 'tch3…'` |
| Share several ports | `tailcat serve 8080,8443` |
| Run a command per connection | `tailcat serve exec -- /path/to/program arg1` |
| Use a SOCKS proxy | `tailcat socks 'tch3…' curl http://server.tailcat:8080/` |
| Keep a server address across restarts | `tailcat genkey --key=default` |
| Read command options | `tailcat --help` or `tailcat <command> --help` |

Saved keys are optional. Once a `default` server key exists, subsequent server runs reuse it; use `--key=new` for a fresh address. See [Security](SECURITY.md) for client allowlists and access control.

## Performance and compatibility

TCP services use reliable QUIC streams; UDP uses QUIC DATAGRAM. Encryption, node authentication and connection secrets are automatic. Public relays can be rate limited, so throughput depends on the path and the machines.

## Experimental features

The browser/WebAssembly demo and browser-inspired TLS fingerprint behavior are described in [Experimental features](docs/experimental.md). They require no setup for normal CLI use. Browser traffic currently uses relays.

## Project

[Build from source](INSTALL.md#build-from-source) · [Changelog](CHANGELOG.md) · [Security](SECURITY.md) · [Contributing and releases](RELEASING.md)

An independent fork, not supported by Tailscale. [BSD-3-Clause license](LICENSE) · [Third-party notices](THIRD_PARTY_NOTICES.md).

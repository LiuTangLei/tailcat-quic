# Tailcat H3

A **QUIC/HTTP/3-only** fork of [tailscale/tailcat v0.6.0](https://github.com/tailscale/tailcat/tree/v0.6.0). Connect two machines using a connection code, without a Tailscale account, control server, root privileges, or system routing changes.

This fork reuses the authenticated H3 transport from [LiuTangLei/tailscale](https://github.com/LiuTangLei/tailscale). **TCP proxy connections use reliable HTTP/3 CONNECT streams**, avoiding a second user-space TCP stack inside QUIC. UDP/IP datagrams use HTTP/3 CONNECT-IP and QUIC DATAGRAM on the same authenticated H3 connection. Neither path wraps WireGuard ciphertext. **BBRv3 is selected by the application on both QUIC endpoints.** Changing the operating system's TCP congestion-control setting does not select the QUIC controller.

The v0.6.0-h3.3 release combines the reliable TCP stream path with validated shared-copy, passive-batching and owned-write-buffer optimizations. The shared experiment improved forward throughput and server memory use, but reverse single-stream throughput decreased by about 15%; loaded latency is not uniformly better. See [the measurements and release-validation boundaries](docs/release-validation-v0.6.0-h3.3.md). This is a proxy tool, not a claim that a general-purpose IP VPN now matches WireGuard on every path.

The executable is still named `tailcat`. Both endpoints should upgrade together to a version advertising the reliable TCP-stream capability. Older H3 peers lacking it receive an explicit incompatibility error, not a WG fallback. Both endpoints must run this H3 fork. Its versioned `tch3…` connection codes are deliberately separate from upstream `tc…` codes. For standard WireGuard tailcat, use the [official project](https://github.com/tailscale/tailcat).

## Install

Download the archive for your operating system and architecture from [Releases](https://github.com/LiuTangLei/tailcat/releases), verify it against `checksums.txt`, and put `tailcat` (`tailcat.exe` on Windows) on your PATH.

```sh
tailcat version
tailcat --help
```

Build from source with the Go version specified in `go.mod`:

```sh
git clone https://github.com/LiuTangLei/tailcat.git
cd tailcat
go build -trimpath -o tailcat ./cmd/tailcat
```

The release uses pinned, published dependency versions. It does not need sibling source directories, a manually supplied certificate, an AWG profile, or transport-selection flags. Use a checked-out release tag for a reproducible source build. Because the module pins transport forks through `replace` directives, building from a checkout is the supported source-install procedure.

## Forward a local service

On the machine running the service:

```sh
tailcat serve 8080
# Copy the generated tch3… connection code privately.
```

On the client:

```sh
tailcat forward 'tch3…' 18080:8080
```

Open `http://127.0.0.1:18080` on the client. Numeric `serve` and `forward` retain their upstream TCP-only CLI semantics; library UDP APIs are separate. Forward listeners bind to loopback by default. Multiple mappings can share a single process:

```sh
tailcat serve 8080,3306
tailcat forward 'tch3…' 18080:8080 13306:3306
```

Use the **actual, complete** generated code in place of `tch3…` in every example.

## Pipe bytes

Run on the receiver:

```sh
tailcat
```

Copy its code, then run on the sender:

```sh
printf 'hello over H3\n' | tailcat 'tch3…'
```

The default receiver accepts one connection and exits after the stream ends. For a persistent network service, use `serve`.

## Check connectivity

```sh
tailcat ping 'tch3…'
tailcat ping --until-direct --timeout=30s 'tch3…'
tailcat --verbose forward 'tch3…' 18080:8080
```

Magicsock provides endpoint discovery and NAT traversal. A direct path and a DERP relay are alternative **paths**, not different tunnel protocols: the data plane remains H3. A successful discovery ping alone is not evidence of an authenticated H3 application-data session. See the release validation report for actual data-transfer coverage.

DERP discovery remains visible to the relay, and a relayed connection still has the outer DERP transport's properties and throughput limits. H3 does not make traffic invisible, reproduce every browser fingerprint, or guarantee connectivity on networks that block QUIC or the selected relay.

## Files and SSH

Share a directory read-only:

```sh
tailcat serve --files=./shared:ro
```

On a client:

```sh
tailcat ls 'tch3…'
tailcat cp 'tch3…:example.txt' ./example.txt
```

For an SSH service with explicit SSH public-key authentication:

```sh
tailcat serve --ssh-authorized-keys=./authorized_keys ssh
tailcat ssh 'tch3…'
```

Use `tailcat serve --help`, `tailcat cp --help`, and `tailcat ssh --help` for the other upstream v0.6 service options. Serving `all`, `exit-node`, writable files, or a shell grants substantial access; expose only what the receiving client needs. In particular, `no-auth-ssh` intentionally relies on tunnel admission rather than SSH authentication and must not be given a public connection code.

## Persistent identity and client restrictions

To preserve a server identity and connection secret across restarts:

```sh
tailcat genkey --key=default
tailcat serve 8080
```

To restrict a server to a particular client identity, generate a client key on that client:

```sh
tailcat genkey --key=client-default --client
```

Copy the printed **public** client key to the server's allowlist:

```sh
tailcat serve --allow='nodekey:CLIENT_PUBLIC_KEY' 8080
```

The client still needs the H3 connection code. Possession of a listed public key is not enough: the client must prove the matching private key and the connection credential. Private key files and connection codes are credentials; never commit them, put them in public DNS, or include them in issue logs.

## Protocol and security

* There is one H3 data plane: reliable CONNECT streams for TCP proxy connections and CONNECT-IP / QUIC DATAGRAM for datagrams. No WG/AWG negotiation or fallback is started. Each TCP CONNECT is accepted only on an already authenticated QUIC session and is checked against the configured service policy.
* The connection code selects the H3 protocol version and contains the server identity, discovery information, and a mandatory random connection secret.
* Client admission requires that secret and respects the optional node allowlist. The H3 handshake additionally binds both node identities to the current TLS session. No trust-on-first-use certificate acceptance or external control-plane identity exchange is required.
* BBRv3 runs in userspace inside the pinned QUIC library, independently for each sending endpoint. A configured controller is not a promise of higher speed on every path or of production-scale congestion fairness.
* The inherited path-discovery and networking dependencies may contain WireGuard compatibility types. Their presence in the dependency graph does not mean a WireGuard data-plane device is instantiated.

Read [SECURITY.md](SECURITY.md) before exposing services. This is an independent fork, not a Tailscale-supported product. Browser/WASM interoperability with upstream is not provided by this native H3 release.

## Development and release validation

```sh
go test -count=1 -timeout=15m ./...
go test -race -p=1 -parallel=1 -count=1 -timeout=20m ./...
go vet ./...
```

CI verifies Linux, macOS, and Windows. Release packaging uses the checked-in build tags and emits checksums. Release notes distinguish runtime-tested platforms and paths from cross-compilation-only checks; finite tests cannot prove the absence of all defects.

## Credits and license

Based on Tailscale's tailcat **v0.6.0**, commit `790406204c002a6f109ef7c6a30a436601a1f6a8`, and the Tailscale open-source networking stack. Original copyright and BSD-3-Clause license are retained in [LICENSE](LICENSE). QUIC, TLS, and other dependencies retain their own licenses and source attribution.

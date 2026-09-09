# Security of the H3 fork

This document describes the independent `LiuTangLei/tailcat` H3 fork. Do not assume upstream WireGuard tailcat's cryptographic design, compatibility, or support policy applies unchanged.

## Report a vulnerability

Use the repository's **Security → Report a vulnerability** facility when it is available. Otherwise, open an issue asking for a private reporting channel without including exploit details, credentials, connection codes, or private infrastructure information. Do not send vulnerabilities specific to this fork to Tailscale as though this were an official product.

## Credentials and admission

An H3 connection code is a **bearer credential**, not a public address. It contains an independent random connection secret as well as the server identity and discovery information. The entire code must be shared through a trusted private channel. Do not publish it in DNS, source code, screenshots, shell transcripts, or issue reports.

Knowing a server's public node key is not sufficient for admission. Client admission must prove possession of the connection secret and satisfy any configured node-key allowlist. The authenticated H3 exchange also proves the participating node identities and binds them to the current TLS connection. A relay that observes public node keys must not be able to join by merely choosing its own client key.

The H3 connection secret is mandatory. The fork must reject an unsupported protocol version, an official `tc…` connection code, and missing or invalid credentials rather than silently opening a less protected transport. Saved identities and their connection metadata are private credentials, including fields with historical names such as `Public` or `PresharedKey`.

Treat `tailcat parse` output as secret: decoding a code does not make its fields safe to publish. The same applies to process arguments, configuration backups, and verbose logs. There is no promise that local administrators cannot observe credentials used on their machine.

## Encryption and trust

TCP proxy connections use reliable HTTP/3 CONNECT streams on an already authenticated QUIC session. A TLS connection alone cannot open a proxy stream: the server checks the exact connection's completed node authentication, current peer authorization, and the embedding application's served-port/forward policy. The connection's presented remote address is derived from the authenticated node identity, not from a client-supplied source address. UDP/IP packets continue to use authenticated CONNECT-IP / QUIC DATAGRAM and source-address checks. There is no WireGuard encryption layer and no WG/AWG data-plane fallback.

Long-lived reliable streams remain on their QUIC connection rather than being truncated by the older IP-only backend's periodic full-connection replacement. QUIC packet-key updates and live peer revocation remain active. Packet-key updates are not the same operation as a new authenticated Diffie-Hellman handshake; this fork does not claim WireGuard's exact rekey schedule or identical cryptographic construction.

TLS identities are created locally and integrated with the existing node-authentication mechanism. This is not public-Web PKI authentication and must not be presented as a publicly trusted HTTPS website. Accepting a provisional certificate is not admission: the transport must complete its bound node proof before admitting application traffic. An implementation change that forwards traffic before authentication is a vulnerability.

Unlike upstream's use of a PSK inside the WireGuard handshake, this fork's connection secret is an admission credential. Do not infer additional post-quantum confidentiality properties merely because that field retains the name `PresharedKey`. Cryptographic properties depend on the negotiated TLS handshake and the actual implementation.

## Scope of the protection

The transport protects application content in transit; it does not hide every characteristic of the connection. Public/private addresses, timing, packet sizes, selected relays, and discovery exchanges may be observable. Real HTTP/3 framing and a browser-style ClientHello do not guarantee indistinguishability from a browser or immunity to active probing, traffic analysis, or blocking.

Direct UDP and DERP relaying are different network paths for the H3 tunnel. A DERP path has its own visible outer transport. Public relays are best-effort services and can be unavailable or rate limited. A discovery ping is not proof that an authenticated application-data tunnel has succeeded.

BBRv3 is a userspace congestion controller, not a security mechanism and not a guarantee of throughput, low latency, or fairness on all networks. Unit tests and a limited WAN test matrix do not replace production-scale congestion-control evaluation.

## Exposed services

Only expose the ports and resources the client needs. `serve all`, `serve exit-node`, writable file service modes, and shell services provide broad access to the server or its network. Keep local forward and SOCKS listeners on loopback unless another machine intentionally needs access to them.

`no-auth-ssh` intentionally does not require a separate SSH credential. Anyone holding an admitted identity and the connection code may receive a shell as the account running tailcat. Prefer the authenticated `ssh` service with an explicit `authorized_keys` source, and use a node allowlist where appropriate. Running as root usually is unnecessary.

Rotate compromised server keys and connection secrets and restart the affected service. Revoking a code does not erase data already obtained by a previously authorized client. Do not reuse upstream key files by assuming that their historical options mean the same thing in this fork.

## Validation and limits

Release validation records the exact tested source and dependencies, tests run, supported build targets, and any limitations. Passing tests is not a guarantee that the implementation is defect-free. Browser/WASM clients from the upstream project are not compatible with this native H3 release.

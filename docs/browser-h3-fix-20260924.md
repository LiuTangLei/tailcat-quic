# Browser H3 fixes — 2026-09-24

Application change: `d360f529b45ccf4334b74386a1cbb261765568d8`.
Base: `f4af70cbe74922e1fe4daa55db435d1dc9ec41d1`.
Branch: `fix/browser-h3-20260924`; no release or dependency pin is changed.

## Two distinct faults

The page loaded WASM successfully, but `newH3Factory` explicitly rejected
GOOS=js with "this native H3 fork does not support browser/WASM clients".
The inherited browser tests waited for a listener that could never appear.
The browser already has a magicsock DERP/WebSocket PacketConn capable of
carrying the same authenticated H3 backend. The unconditional platform
rejection was removed, not replaced with WireGuard or unauthenticated data.

After enabling this path, browser-to-native file/text sending passed, but
native-to-browser receiving still failed. The H3 backend closes a stream
when its OnTCP handler returns. The WASM handler returned immediately after
calling JavaScript, even though JavaScript had only scheduled asynchronous
reads. It now keeps that handler alive until the connection's existing Done
signal closes, just as an accepted native listener transfers ownership.
JavaScript Close or session shutdown releases the handler.

The connection secret requirement, TLS identity and session-bound node proof,
live peer policy, H3 framing and reliable stream transport are unchanged.
Browser traffic is relayed over WebSockets; no browser raw-UDP capability or
direct peer-to-peer browser path is claimed.

The documented WASM build now supplies `-o tailcat-web.wasm`, avoiding a
collision with the existing `web/` directory.

## Verification

A new real-Chrome bootstrap check fails quickly on page startup errors and
reports readiness and H3-address presence, not the connection capability.
The initial baseline produced the explicit platform rejection. Merely
removing the guard fixed startup and browser sending, but not receiving.
Those intermediate failed receive logs are retained locally.

After the lifetime fix, bootstrap, binary receive/send and text receive/send
all passed against the existing published QUIC dependency. Then the same five
real-Chrome tests ran twice with shared QUIC candidate `7ca60b742f2f` inherited
by both Go tests and the WASM subprocess via an external modfile: **10/10**
passed, with binary transfers checking 2 MiB per direction and SHA-256.

Full native application/CLI/web tests and the serialized race suite also
passed with the candidate library: 266 test/subtest events in each, with the
five opt-in browser tests reported separately. Authentication/negative-key
checks remain in the existing suites; no error or reset was converted into
successful delivery to obtain these results.

No official go.mod/go.sum was changed. The raw browser logs contain ephemeral
test capabilities and are private. This validates local headless Chrome,
not Safari, Firefox, every browser version, WAN performance or a multi-hour
browser memory soak. Existing release assets were not replaced.

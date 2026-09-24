# Tailcat-QUIC v0.7.0-quic.3 — validation scope

## Exact runtime inputs

- Tailcat browser fixes: `d360f529b45ccf4334b74386a1cbb261765568d8`, documented at `c56532bfbf58d4dec21098c7ada7b4c69b6ed080`.
- QUIC: `github.com/LiuTangLei/quic-go v0.63.0-quic.3`, source `3d9e7d1f348241b89ec1060f733ea39b967e7f82`; recovery code `7ca60b742f2f23765ef3cce85729e5103289147f`.
- Tailscale integration remains `github.com/LiuTangLei/tailscale v1.102.5-0.20260923003659-b4f1aa3cd300`.
- WG/TUN primitives remain `github.com/LiuTangLei/wireguard-go v0.0.33-0.20260910045057-ed22747d204e`.

No local module replacement is permitted in a release package. The source commit, resolved public modules, toolchain and hashes are recorded in the accompanying `build.json`.

## Recorded code validation

QUIC ordinary and full serial race tests passed after the recovery changes. Tailcat application/CLI/web tests and race tests passed. Real Chrome listener/file/text tests passed against the candidate library; browser transport remains authenticated QUIC/H3 carried by DERP/WebSocket, not browser direct UDP and not a fallback to WireGuard.

The controlled recovery matrix completed 16 cases of 200 MiB with matching length and SHA-256. In the subsequent September 24 check, a 100-to-20 Mbps capacity drop completed rather than aborting; 20 Mbps / 60 ms RTT / 5% random loss completed at approximately 14.30 Mbps, and the 200 ms RTT reverse case completed at approximately 10.90 Mbps. These are virtual-link results, not Internet performance measurements. The capacity-drop whole-transfer average includes the initial faster period and is not its post-drop steady-state rate.

Healthy-path local socket comparisons did not demonstrate lower whole-process CPU. Avoid describing this release as a universal throughput or CPU improvement. The clearly measured improvement is reliable recovery in the tested adverse conditions.

## Final package gate

The paired QUIC/Tailcat release preparation on September 25 rechecks exact source identity, builds only from public immutable pins, retains all seven existing executable archive targets and six Linux packages, verifies checksums and tests executables extracted from the actual archives. Current packaging additionally includes all three README languages. Final package outcomes are recorded in the separate `VALIDATION.md` asset rather than implied by this source document.

The requested remote preflight was blocked by the execution tool before commands ran on the Japan hosts. It was not reissued through another route. Accordingly, both releases stay in draft; previous public Latest releases, production services and default installer aliases are unchanged. No new real-host throughput, CPU, NAT matrix, Safari/Firefox, Windows/Android device or multi-hour endurance claim is made.

Native local Linux/container checks, when present in `VALIDATION.md`, must not be relabeled as WAN tests. A successfully cross-compiled binary is not a native platform runtime test.

## Promotion

After the real-host gate succeeds, verify the draft asset readback, publish both releases, and update installer metadata from the exact uploaded archive hashes. Do not rewrite tags, regenerate different files under the same published filenames, or move stable install aliases before the assets are publicly available. The shared QUIC repository must remain automation-free.

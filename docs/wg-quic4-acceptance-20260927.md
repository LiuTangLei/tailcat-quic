# quic.4 acceptance and WireGuard comparison — 2026-09-27

## Result and method

The browser fixes and the recorded functional/package checks pass within the scope below. QUIC reached 556.35 / 505.77 Mbps with four parallel TCP streams from jp-6c12g to jp1010; this is the only QUIC case where both 20-second receiver averages exceed 500 Mbps. Its single-stream result was 504.42 / 471.85 Mbps, versus WG 515.04 / 525.60 Mbps. The other direction and the other tested pairs did not reach the target. This supports a limited four-stream result, not stable single-stream/bidirectional 500 Mbps or universal superiority over WG.

Five Linux servers, three pairs; each pair ran WG → QUIC → QUIC → WG, with single and four parallel TCP streams in both directions. Each sample measured 20 seconds after two omitted warm-up seconds. Tests sharing an endpoint were serialized. The EPYC pair was added after the initial Xeon pair measured below 500 Mbps; every initial sample is retained. SSH used the management overlay only. Japanese tunnel endpoints automatically selected direct IPv6 and sometimes also observed IPv4; AU/US observed IPv4. The initial native baselines used public IPv4, so they are not evidence of a same-path Japanese tunnel ceiling. A public IPv6 native baseline and 500 Mbps UDP checks were subsequently added for the EPYC pair. No per-sample fixed IP-family assertion is made from the endpoint log sets.

WG is the official Tailcat v0.7.0 userspace WireGuard executable, source `15ab9e68bfc6534a61797d7af28cedd42b54a3a5`. Both executables use Go 1.27.1, but dependency versions/build tags differ: this is a comparison of the two application implementations, not kernel WireGuard or an isolated cipher benchmark. Both endpoint logs confirm their expected engine and direct transport.

## All tunnel samples

Receiver Mbps; cells are round 1 / round 2. Four streams are their aggregate in one direction, never the sum of opposite directions.

| Direction | Streams | WG | QUIC | QUIC vs WG mean | QUIC ≥500 in both rounds |
| --- | ---: | ---: | ---: | ---: | --- |
| jp1010 → 4t-jp | 1 | 440.27 / 449.60 | 447.51 / 437.23 | -0.6% | no |
| 4t-jp → jp1010 | 1 | 406.70 / 406.65 | 329.62 / 313.36 | -20.9% | no |
| jp1010 → 4t-jp | 4 | 470.66 / 454.07 | 456.29 / 446.57 | -2.4% | no |
| 4t-jp → jp1010 | 4 | 420.85 / 417.98 | 330.68 / 342.00 | -19.8% | no |
| jp1010 → jp-6c12g | 1 | 256.67 / 271.66 | 243.54 / 251.62 | -6.3% | no |
| jp-6c12g → jp1010 | 1 | 515.04 / 525.60 | 504.42 / 471.85 | -6.2% | no |
| jp1010 → jp-6c12g | 4 | 262.26 / 262.13 | 252.38 / 253.18 | -3.6% | no |
| jp-6c12g → jp1010 | 4 | 476.93 / 478.54 | 556.35 / 505.77 | +11.2% | yes |
| us1420 → au9929 | 1 | 99.06 / 135.39 | 269.81 / 300.67 | +143.3% | no |
| au9929 → us1420 | 1 | 69.44 / 19.78 | 218.33 / 251.66 | +426.8% | no |
| us1420 → au9929 | 4 | 165.90 / 221.52 | 358.58 / 335.34 | +79.1% | no |
| au9929 → us1420 | 4 | 114.00 / 116.88 | 239.74 / 253.39 | +113.6% | no |

## Native baseline

| Pair | Direction | Streams | TCP receiver Mbps | Retransmits |
| --- | --- | ---: | ---: | ---: |
| jp-jp | jp1010 → 4t-jp | 1 | 6164.65 | 1715 |
| jp-jp | 4t-jp → jp1010 | 1 | 5961.90 | 477 |
| jp-jp | jp1010 → 4t-jp | 4 | 6261.95 | 3181 |
| jp-jp | 4t-jp → jp1010 | 4 | 6365.32 | 1306 |
| jp-epyc | jp1010 → jp-6c12g | 1 | 4438.21 | 331 |
| jp-epyc | jp-6c12g → jp1010 | 1 | 6218.56 | 2918 |
| jp-epyc | jp1010 → jp-6c12g | 4 | 5071.78 | 1288 |
| jp-epyc | jp-6c12g → jp1010 | 4 | 7170.20 | 11961 |
| au-us | us1420 → au9929 | 1 | 162.36 | 0 |
| au-us | au9929 → us1420 | 1 | 137.12 | 114 |
| au-us | us1420 → au9929 | 4 | 466.48 | 3 |
| au-us | au9929 → us1420 | 4 | 566.55 | 112 |

The table above is IPv4. Supplemental EPYC IPv6 TCP receiver rates were 4957.46 / 6214.12 Mbps (single stream, forward/reverse) and 5350.16 / 7188.99 Mbps (four streams). IPv6 UDP with a requested 500M pacing rate received 321.66 Mbps with 35.78% loss forward, and 260.25 Mbps with 0.31% loss reverse. IPv4 UDP results and all exact counters are retained in the JSON. UDP 500M is the requested pacing rate, not proof that the generator sent at that rate. These are not passing 500 Mbps low-loss tunnel-UDP tests. Native results are not a universal link ceiling, and these measurements alone do not locate a CPU, packet-generation or congestion-control bottleneck. No system routes, firewall, congestion sysctls or production services were changed.

## Functional and package acceptance

All 12 tunnel runs passed length/SHA-256 verification (509,620,224 bytes), concurrent transfer and recovery after idle. All 48 throughput samples are retained. Loaded echo probes reported zero errors; no endpoint panic or owned-process cleanup error was reported. Post-run checks on all five hosts found no running test executables or native baseline listeners; eight pre-existing test directories dated September 10/23 were left intact. This WG comparison did not test tunneled UDP. Earlier QUIC-specific validation recorded 420/420 UDP probes and retained its original failure/rerun evidence in `browser-lifecycle-fix-20260927.md`.

The browser runtime files, tests, module pins and build tags still match the hashes of the previously passed Linux ordinary/race/vet, WASM/JavaScript, six Chrome and macOS native/Chrome checks. No native runtime code was changed for these benchmarks.

Clean package source: `bf44335b5cc98aba21b1732246835d19a9d35866`. Seven executable archives and three DEB/three RPM packages were built with public immutable module pins. All 13 hashes passed, all 13 embedded executables match the corresponding build manifest, all seven archives contain the required documentation, and every embedded executable has the expected dependency/source identity. Linux CLI end-to-end tests passed using the executable extracted from the actual archive.

WAN measurements used Linux executable SHA-256 `c8784ae1a95d1cec0cfb81bfc9ccafcc160ee6ca59553fdfa4fdd866b9559320`, built from the quic.3 base plus the recorded fixed source hashes. The final clean-commit Linux package executable is `62fa9c39c775a4c9ba75ade0a31466f2db73e8711cb0c66ea753fa62237f3cab`; these are separately identified artifacts with matching runtime source, not an assertion that their bytes are identical.

The final packaged macOS/Windows/ARM executables were cross-compiled and inspected; final macOS archive execution, Windows/Android device testing, Safari/Firefox and multi-hour endurance are outside this run. Earlier macOS execution used the fixed test candidate. No release/tag was published or installer alias changed.

## Evidence

- Machine-readable samples, engine evidence, CPU/RSS, latency, integrity, baseline identity, package hashes and dependency revisions: `validation/wg-quic4-acceptance-20260927.json`.
- Candidate packages and verification logs: `/workspace/tailcat-quic-release-20260927/`.
- Raw remote results and bounded runners: Mac `audits/wg-quic4-acceptance-20260927/`.
- Earlier regression failures and reruns: `browser-lifecycle-fix-20260927.md` and `validation/browser-lifecycle-20260927.json`.

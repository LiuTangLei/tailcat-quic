# Tailcat-QUIC project agreements

- This application lives only in LiuTangLei/tailcat-quic. Do not create duplicate Tailcat application or full-library mirror repositories.
- New application releases use ordinary semantic versions or a `quic.N` suffix (for example `v0.7.0-quic.2`). Never create another `h3.N` release. Existing historical tags are immutable.
- GitHub Actions may be used in this public application repository only on standard GitHub-hosted runners, for bounded release/package jobs that are free for public repositories. Do not use larger/GPU runners, long-lived artifacts/caches, Dependabot schedules, or per-commit heavy builds. The shared `quic-go` repository remains automation-free and must not regain `.github` on upstream merges.
- Local tests and real-host performance validation remain the release gate. Cloud packaging must consume an already reviewed tag/commit; it is not a substitute for WAN validation. Publication remains a separate explicit step after package verification.
- QUIC/H3-only transport, BBRv3, mandatory connection secret and node authentication remain. Do not replace reliable TCP streams with TCP-over-DATAGRAM or change cipher/authentication rules as a throughput shortcut.
- Preserve bounded ownership, graceful final-byte/FIN delivery and immediate explicit abort/revocation. Benchmark both directions; retain failures and slower samples.
- Use public immutable module pins in releases, not local replacements, and record exact dependency revisions, artifact hashes and test scope.

# Tailcat-QUIC project agreements

- This application lives only in LiuTangLei/tailcat-quic. Do not create duplicate Tailcat application or full-library mirror repositories.
- New application releases use ordinary semantic versions or a `quic.N` suffix (for example `v0.7.0-quic.2`). Never create another `h3.N` release. Existing historical tags are immutable.
- Do not add a `.github` directory, Actions workflow, Dependabot schedule or automatic cloud build. GitHub Actions is disabled by the owner. Upstream merges must not restore these files.
- Build, test and package locally or on explicitly selected test hosts. Publishing is a separate manual action after real executable checks and performance measurements; no unverified background publication.
- QUIC/H3-only transport, BBRv3, mandatory connection secret and node authentication remain. Do not replace reliable TCP streams with TCP-over-DATAGRAM or change cipher/authentication rules as a throughput shortcut.
- Preserve bounded ownership, graceful final-byte/FIN delivery and immediate explicit abort/revocation. Benchmark both directions; retain failures and slower samples.
- Use public immutable module pins in releases, not local replacements, and record exact dependency revisions, artifact hashes and test scope.

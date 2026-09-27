# Contributing and releasing

Application source and packages are maintained in [LiuTangLei/tailcat-quic](https://github.com/LiuTangLei/tailcat-quic). Start from the `quic-v0.7` branch.

For a code change, run the relevant tests and include a short explanation of the behavior and validation. Networking changes need both-direction integrity and performance evidence; retain failures and slower samples. Browser changes need real-browser checks as well as a successful WASM build.

Use the [release procedure](docs/manual-release.md) to build, verify and publish. New versions use semantic versions or a `quic.N` suffix. Historical tags are immutable.

For the current release scope, see [quic.4 validation](docs/release-validation-v0.7.0-quic.4.md) and [the WG comparison](docs/wg-quic4-acceptance-20260927.md).

# Contributing and releasing

Application source and packages are maintained in [LiuTangLei/tailcat-quic](https://github.com/LiuTangLei/tailcat-quic). Start from the branch for the release you are preparing.

For a code change, run the relevant tests and include a short explanation of the behavior and validation. Networking changes need both-direction integrity and performance evidence; retain failures and slower samples. Browser changes need real-browser checks as well as a successful WASM build.

Use the [release procedure](docs/manual-release.md) to build, verify and publish. New versions use semantic versions or a `-quic` suffix. Historical tags are immutable.

For the current release scope, see [v0.8.0-quic validation](docs/release-validation-v0.8.0-quic.md).

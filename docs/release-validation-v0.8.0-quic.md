# v0.8.0-quic validation

This page records validation performed on the candidate before release packaging. Final source, real-host measurements and artifact verification are recorded in the accompanying `VALIDATION.md` and `build.json`.

## Candidate tested

- Application: `release/v0.8.0-quic` at `d2bfbe29e` with the local admission and release changes.
- Core: `e5716e2c4d6618f1f2695b32baf888bdf5ccb7e6`. Candidate tests used that checkout. The release source pins the same commit as `github.com/LiuTangLei/tailscale v1.104.7-tailcat.1` with its public checksum. QUIC: `github.com/LiuTangLei/quic-go v0.65.0-quic.4`.
- Go 1.27.4 on macOS ARM64, with `GOWORK=off`. Local integration tests disabled port mapping to isolate the host LAN.

## Results

- Tailcat's Go test suite with `-race`, `go vet ./...`, and `go mod verify` passed.
- The pending-approval revocation test failed against the original application source and passed 20 consecutive runs with `-race` after the fix.
- Core race tests passed in `wgengine/magicsock`, `wgengine/netstack`, `net/netmon`, `net/netns`, and `feature/derpwebsocket`.
- Six Chrome integration tests passed, including WASM transfer integrity in both directions and repeated transfers over a local DERP fixture.
- The `v0.8.0-quic` test build succeeded for Linux amd64, arm64 and armv7, macOS amd64 and arm64, and Windows amd64 and arm64. The native macOS ARM64 binary reported `v0.8.0-quic` and passed the CLI suite.

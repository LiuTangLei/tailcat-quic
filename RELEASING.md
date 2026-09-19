# Releasing Tailcat H3

This repository is the independent QUIC/H3-only fork. Release tags use `v0.6.0-h3.N` for the upstream v0.6 baseline. Do not run the upstream `tag.sh` blindly: publishing this fork also requires the authenticated H3 and QUIC controller dependency gates below.

## Dependency gate

Publish immutable versions of the QUIC and Tailscale H3 dependencies first, after their relevant unit, integration, race, and integrity tests pass. Keep existing upstream/default congestion policies unchanged. Pin published module versions in this repository's `go.mod`; local replacements and build-time source overlays are forbidden in a release.

The current transport dependency pins are:

- `github.com/LiuTangLei/tailcat-quic-go v0.62.0-tailcat.4`
- `github.com/LiuTangLei/tailcat-tailscale v1.102.4-tailcat.1`

The upstream Go import paths are intentionally retained through module replacements. Build this CLI from a checked-out tag rather than using `go install …@version`, which does not support a main module's dependency replacements.

## Validation gate

Run `go mod tidy`, `go mod verify`, `go vet ./...`, the full test suite, and the race suite. Under race instrumentation, serialize heavyweight operating-system/network fixtures with `-p=1 -parallel=1`; the tests themselves still exercise concurrent streams and callbacks. Do not skip functional tests to create a release.

```sh
go mod tidy
go mod verify
go test -count=1 -timeout=15m ./...
go test -race -p=1 -parallel=1 -count=1 -timeout=20m ./...
go vet ./...
go test -run '^$' -fuzz '^FuzzH3ConnectionCode$' -fuzztime=10s -parallel=2 .
```

Run real application-data integrity tests, not just discovery pings. `scripts/wan_smoke.py` uses temporary loopback fixtures and separate CLI processes; it accepts the test host at runtime so private infrastructure is not committed. Exercise both direct UDP and forced DERP paths. It supports a RAM-backed temporary root on a test machine whose persistent disk is full, but never clears unrelated files or changes production services.

Record exact tested conditions in `docs/release-validation-v0.6.0-h3.1.md` or the report for the next release. Distinguish runtime testing from cross-compilation. Never claim tests prove the absence of all bugs or guarantee network performance.

## Publication gate

Push the branch and check its **Test** workflow. Only tag the intended reviewed commit after the checks pass. The **Release** workflow reruns the test matrix and cannot package until it succeeds. It creates a **draft** release, not a published notification.

Before publishing the draft, verify:

- The tag is rooted in the documented upstream version and all module pins are public.
- Every expected executable archive and Linux package is attached, together with `checksums.txt`.
- Downloaded executable archives match their checksums and contain the expected version and dependency build information.
- Native runtime smoke tests use the packaged binaries, not merely an earlier development binary.
- Release notes and bundled documentation describe actual test coverage and do not contain connection codes, private server addresses, or local worktree paths.

Publishing the draft is the final maintainer action. Do not move a published tag to repair a problem; publish a new H3 revision instead.

## Artifacts

Release executables: Linux amd64/arm64/armv7, macOS amd64/arm64, Windows amd64/arm64. Archives use tar.gz, except Windows uses zip. Linux DEB/RPM packages are named `tailcat-h3` and install the `tailcat` executable, conflicting with the standard `tailcat` package rather than silently co-installing two different programs under the same name.

The fork does not publish to Tailscale's container registry, Homebrew formula, browser demo, or Nix package. The executable build tags in `.goreleaser.yaml` and `build-tags.txt` are enforced by the existing build-tag synchronization tests.

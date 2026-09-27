# Release procedure

## Source and dependencies

Use a clean commit on the application branch `quic-v0.7`, with public immutable module pins and no local dependency replacements. New versions use semantic versions or a `quic.N` suffix; preserve historical tags. Record the exact source revision, dependency revisions, toolchain, test scope and artifact hashes.

Keep maintenance policy here and in `AGENTS.md`, rather than in the user quick start. The shared `quic-go` repository remains automation-free. This public application repository may use bounded, manually dispatched package/container jobs on standard GitHub-hosted runners; do not add heavier per-commit automation, larger/GPU runners or long-lived caches/artifacts.

## Validate and package

Run relevant ordinary, race and vet checks. Runtime changes require local application tests and real-host tests in both directions, including integrity, final-byte/FIN delivery, explicit abort, direct and relay paths. Browser changes need real-browser validation. Preserve failures and slower samples. A cross-build is not native execution, and cloud packaging does not replace WAN validation.

For example:

```sh
go mod verify
go test -count=1 -timeout=5m ./...
go test -race -p=1 -parallel=1 -count=1 -timeout=5m ./...
go vet ./...
python3 scripts/local-package.py --version v0.7.0-quic.4 \
  --output /absolute/path/outside/checkout/release --linux-packages
python3 scripts/verify-release-assets.py --tag v0.7.0-quic.4 \
  --assets /absolute/path/outside/checkout/release --revision FULL_COMMIT_ID
```

The package script produces seven executable archives and six Linux packages. `build.json` records source/toolchain/modules/hashes; `checksums.txt` covers the distributable packages. Test-only builds are never release assets. Verify documentation and binary identity inside the packages, and exercise actual native executables on available systems.

## Publish

After publication is authorized and validation is complete:

1. Push the reviewed source branch and create an immutable application tag.
2. Create a draft release with the 13 packages, checksums, sanitized build metadata and validation record.
3. Download the uploaded assets, compare every hash and verify extracted executables before publishing the release as Latest.
4. Update the default-branch shell/PowerShell installer versions, `packaging/release.json`, and Scoop metadata using the exact published hashes. Homebrew and Nix consume `packaging/release.json`.
5. Resolve the published application module checksum, update the nested `install` module's version/checksum and create its `install/vX.Y.Z-quic.N` tag. Check the documented Go bootstrap.
6. Dispatch the bounded container job with the explicit verified release version and check both image platforms and native execution.
7. Read back the public release, default-branch install metadata and at least the available native install paths. Do not rewrite tags or replace published assets with different bytes.

Only switch stable installer metadata after the corresponding assets are public. Subsequent metadata-only commits do not change the source identity recorded for the already built release.

# Manual local release

GitHub Actions is intentionally disabled. Do not recreate .github or invoke a
hosted workflow. Public standard runners being free is not permission to turn
automatic builds back on.

Use a clean checkout and public immutable dependencies. New application tags
are ordinary versions or `quic.N`; old `h3.N` tags are historical only.

```sh
go mod verify
go test -count=1 -timeout=5m ./...
go test -race -p=1 -parallel=1 -count=1 -timeout=5m ./...
go vet ./...
python3 scripts/local-package.py --version v0.7.0-quic.2 \
  --output /absolute/path/outside/checkout/release --linux-packages --udp-probe
python3 scripts/verify-release-assets.py --tag v0.7.0-quic.2 \
  --assets /absolute/path/outside/checkout/release
```

The build script writes seven executable archives and, when requested, three
deb plus three rpm packages using pinned nFPM v2.33.1. It does not modify source,
install services, change network settings, contact release APIs or publish.
`build.json` records source revision, toolchain, selected modules and hashes.
`checksums.txt` covers all distributable packages. Test-only local dependency
builds are explicitly marked and never archived as release packages.

Verify actual native executables on available operating systems and test the
public-pin Linux executable through direct and forced-relay paths. Record test
coverage honestly; a cross-build is not a native runtime test. Preserve SHA-256
integrity, UDP/stream boundaries and final-byte/close checks. WAN failures or slow
samples must not be silently excluded.

After verification, create an immutable tag, push the chosen source branch and
tag, create a GitHub Release draft and upload only the named archives, packages,
checksums and sanitized build/validation records. Download/read back uploaded
assets before removing draft status. Never enable Actions for this step. A
published tag alone is not a completed binary release.

Use `quic-v0.7` as the active 0.7 application branch. Do not force-push or rewrite
historical tags. The shared QUIC library follows the same no-Actions policy;
its `.github` directory must stay absent during upstream merges.

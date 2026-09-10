#!/usr/bin/env python3
"""Verify checksums, native executable identity, and CLI behavior of a draft release.

Requires gh, Go and Python 3.9+. Credentials come only from gh's environment.
Archives and extracted executables live in a private temporary directory, never
in the source checkout. This command does not publish or edit a release.
"""
import argparse
import hashlib
import os
from pathlib import Path
import platform
import re
import subprocess
import tarfile
import tempfile
import zipfile


def run(command, **kwargs):
    return subprocess.run(command, check=True, text=True, **kwargs)


def verify(assets, tag, revision):
    checksums = {}
    for line in (assets / "checksums.txt").read_text().splitlines():
        digest, name = line.split(maxsplit=1)
        name = name.lstrip("*")
        if not re.fullmatch(r"[0-9a-f]{64}", digest) or Path(name).name != name:
            raise ValueError("unsafe or malformed checksum entry")
        if name in checksums:
            raise ValueError("duplicate checksum entry")
        checksums[name] = digest
    version = tag.removeprefix("v")
    expected = {
        "tailcat_" + version + "_" + target + extension
        for target, extension in [
            ("linux_amd64", ".tar.gz"), ("linux_arm64", ".tar.gz"),
            ("linux_armv7", ".tar.gz"), ("darwin_amd64", ".tar.gz"),
            ("darwin_arm64", ".tar.gz"), ("windows_amd64", ".zip"),
            ("windows_arm64", ".zip")]
    }
    if not expected.issubset(checksums):
        raise ValueError("release is missing one or more executable archives")
    if len([n for n in checksums if n.endswith(".deb")]) != 3:
        raise ValueError("release must include three Debian packages")
    if len([n for n in checksums if n.endswith(".rpm")]) != 3:
        raise ValueError("release must include three RPM packages")
    for name, expected_digest in checksums.items():
        actual = hashlib.sha256((assets / name).read_bytes()).hexdigest()
        if actual != expected_digest:
            raise ValueError("SHA-256 mismatch: " + name)
    print("Verified SHA-256 for", len(checksums), "release assets", flush=True)

    system = {"Linux": "linux", "Darwin": "darwin", "Windows": "windows"}[platform.system()]
    arch = {"x86_64": "amd64", "amd64": "amd64", "arm64": "arm64", "aarch64": "arm64"}[platform.machine().lower()]
    extension = ".zip" if system == "windows" else ".tar.gz"
    archive = assets / ("tailcat_" + version + "_" + system + "_" + arch + extension)
    executable_name = "tailcat.exe" if system == "windows" else "tailcat"
    required = {"LICENSE", "README.md", "SECURITY.md", "THIRD_PARTY_NOTICES.md"}
    with tempfile.TemporaryDirectory(prefix="tailcat-release-extracted-") as extracted:
        executable = Path(extracted) / executable_name
        # Read just the named regular executable; never trust archive paths,
        # links or permission metadata when extracting an external archive.
        if extension == ".zip":
            with zipfile.ZipFile(archive) as packed:
                names = {Path(n).name for n in packed.namelist()}
                if not required.issubset(names):
                    raise ValueError("archive is missing documentation or license notices")
                matches = [n for n in packed.namelist() if n == executable_name]
                if len(matches) != 1:
                    raise ValueError("archive must have exactly one root executable")
                executable.write_bytes(packed.read(matches[0]))
        else:
            with tarfile.open(archive, "r:gz") as packed:
                names = {Path(m.name).name for m in packed.getmembers() if m.isfile()}
                if not required.issubset(names):
                    raise ValueError("archive is missing documentation or license notices")
                matches = [m for m in packed.getmembers() if m.name == executable_name and m.isfile()]
                if len(matches) != 1:
                    raise ValueError("archive must have exactly one root executable")
                with packed.extractfile(matches[0]) as source:
                    executable.write_bytes(source.read())
        executable.chmod(0o700)
        output = run([str(executable), "version"], capture_output=True).stdout.strip()
        if output != tag:
            raise ValueError("packaged executable has the wrong version: " + output)
        info = run(["go", "version", "-m", str(executable)], capture_output=True).stdout
        for marker in ["github.com/LiuTangLei/quic-go\tv0.62.0-tailcat.3",
                       "github.com/LiuTangLei/tailscale\tv1.102.3-tailcat.3",
                       "vcs.revision=" + revision, "vcs.modified=false"]:
            if marker not in info:
                raise ValueError("packaged build identity is missing: " + marker)
        if "tailcat_perf" in info or "/Users/lei" in info:
            raise ValueError("package contains diagnostic tags or a local dependency path")
        print("Verified native package:", system, arch, output, revision, flush=True)
        env = os.environ.copy()
        env["GOWORK"] = "off"
        env["TAILCAT_TEST_BINARY"] = str(executable)
        run(["go", "test", "-count=1", "-timeout=5m", "./cmd/tailcat"], env=env, timeout=360)
        print("Packaged CLI end-to-end tests passed", flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--repo", default="LiuTangLei/tailcat-quic")
    parser.add_argument("--assets", type=Path, help="verify an existing download instead of downloading")
    args = parser.parse_args()
    if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?", args.tag):
        parser.error("unexpected release tag")
    revision = run(["git", "rev-parse", "HEAD"], capture_output=True).stdout.strip()
    if args.assets:
        verify(args.assets.resolve(), args.tag, revision)
    else:
        with tempfile.TemporaryDirectory(prefix="tailcat-release-assets-") as directory:
            run(["gh", "release", "download", args.tag, "--repo", args.repo, "--dir", directory], timeout=180)
            verify(Path(directory), args.tag, revision)


if __name__ == "__main__":
    main()

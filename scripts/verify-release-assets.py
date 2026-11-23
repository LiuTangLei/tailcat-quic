#!/usr/bin/env python3
"""Verify checksums, native executable identity, and CLI behavior of a draft release.

Requires gh, Go and Python 3.9+. Credentials come only from gh's environment.
Archives and extracted executables live in a private temporary directory, never
in the source checkout. This command does not publish or edit a release.
"""
import argparse
import hashlib
import json
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


def release_module_pins(revision):
    names = (
        "tailscale.com",
        "github.com/quic-go/quic-go",
        "github.com/refraction-networking/utls",
        "github.com/LiuTangLei/wireguard-go",
    )
    env = {**os.environ, "GOWORK": "off"}
    pins = []
    with tempfile.TemporaryDirectory(prefix="tailcat-release-modules-") as directory:
        for filename in ("go.mod", "go.sum"):
            source = run(["git", "show", f"{revision}:{filename}"], capture_output=True).stdout
            (Path(directory) / filename).write_text(source)
        for name in names:
            result = run(["go", "list", "-mod=readonly", "-m", "-json", name],
                         cwd=directory, env=env, stdout=subprocess.PIPE)
            module = json.loads(result.stdout)
            selected = module.get("Replace") or module
            if not selected.get("Version") or not selected.get("Sum"):
                raise ValueError("release dependency lacks a public version or checksum: " + name)
            pins.append(("=>" if "Replace" in module else "dep",
                         selected["Path"], selected["Version"], selected["Sum"]))
    return pins


def verify(assets, tag, revision, pins):
    env = {**os.environ, "GOWORK": "off"}
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
    required = {"LICENSE", "README.md", "INSTALL.md", "SECURITY.md", "THIRD_PARTY_NOTICES.md", "release-validation-" + tag + ".md"}
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
        version_output = run(["go", "version", "-m", str(executable)], env=env, capture_output=True).stdout
        # The first line names the extracted executable, not a build dependency.
        info = "\n".join(version_output.splitlines()[1:])
        # Go indents replacement records an extra tab within each dep record.
        records = {tuple(line.lstrip("\t").split("\t")) for line in info.splitlines()}
        for pin in pins:
            if pin not in records:
                raise ValueError("packaged dependency does not match release source: " + pin[1])
        if ("build", "vcs.revision=" + revision) not in records or ("build", "vcs.modified=false") not in records:
            raise ValueError("packaged source revision does not match clean release source")
        local = any(record[0] == "=>" and (len(record) != 4 or not record[3].startswith("h1:"))
                    for record in records)
        if local or "tailcat_perf" in info or "/Users/lei" in info:
            raise ValueError("package contains diagnostic tags or a local dependency path")
        print("Verified native package:", system, arch, output, revision, flush=True)
        env["TS_DISABLE_PORTMAPPER"] = "true"
        env["TAILCAT_TEST_BINARY"] = str(executable)
        run(["go", "test", "-count=1", "-timeout=5m", "./cmd/tailcat"], env=env, timeout=360)
        print("Packaged CLI end-to-end tests passed", flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--repo", default="LiuTangLei/tailcat-quic")
    parser.add_argument("--assets", type=Path, help="verify an existing download instead of downloading")
    parser.add_argument("--revision", help="exact source commit used to build the executable")
    args = parser.parse_args()
    if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?", args.tag):
        parser.error("unexpected release tag")
    revision = run(["git", "rev-parse", "--verify", (args.revision or "HEAD") + "^{commit}"], capture_output=True).stdout.strip()
    pins = release_module_pins(revision)
    if args.assets:
        verify(args.assets.resolve(), args.tag, revision, pins)
    else:
        with tempfile.TemporaryDirectory(prefix="tailcat-release-assets-") as directory:
            run(["gh", "release", "download", args.tag, "--repo", args.repo, "--dir", directory], timeout=180)
            verify(Path(directory), args.tag, revision, pins)


if __name__ == "__main__":
    main()

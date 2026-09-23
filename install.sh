#!/bin/sh
# Install the QUIC fork only. No sudo, service startup, keys or firewall changes.
# Usage: curl -fsSL .../install.sh | sh
#        sh install.sh --version v0.7.0-quic.2 --bin-dir /custom/bin
set -eu

tailcat_install() {
    version=${TAILCAT_VERSION:-v0.7.0-quic.2}
    bindir=${TAILCAT_BIN_DIR:-}
    dry_run=false
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --version|--bin-dir)
                [ "$#" -ge 2 ] || { echo "Missing value for $1" >&2; return 2; }
                case "$1" in --version) version=$2;; --bin-dir) bindir=$2;; esac
                shift 2 ;;
            --dry-run) dry_run=true; shift ;;
            -h|--help) echo 'Usage: install.sh [--version vX.Y.Z-quic.N] [--bin-dir PATH] [--dry-run]'; return 0 ;;
            *) echo "Unknown argument: $1" >&2; return 2 ;;
        esac
    done
    printf '%s\n' "$version" | LC_ALL=C grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-quic\.[0-9]+)?$' || {
        echo 'Invalid QUIC release version.' >&2; return 2;
    }
    case "$(uname -s)" in
        Linux) os=linux ;;
        Darwin) os=darwin ;;
        FreeBSD|OpenBSD)
            echo 'Use the source installer: go run github.com/LiuTangLei/tailcat-quic/install@latest' >&2
            return 2 ;;
        *) echo 'Unsupported OS; Windows users should use install.ps1 or Scoop.' >&2; return 2 ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch=amd64 ;;
        aarch64|arm64) arch=arm64 ;;
        armv7*|armv8l) arch=armv7 ;;
        *) echo 'Unsupported architecture: no matching published binary.' >&2; return 2 ;;
    esac
    [ "$os/$arch" != darwin/armv7 ] || { echo 'No 32-bit macOS release.' >&2; return 2; }
    if [ -z "$bindir" ]; then
        if [ -n "${PREFIX:-}" ] && [ -d "${PREFIX}/bin" ]; then
            bindir=$PREFIX/bin
        elif [ "$(id -u)" = 0 ]; then
            bindir=/usr/local/bin
        else
            bindir=${HOME:?HOME is required}/.local/bin
        fi
    fi
    case "$bindir" in /*) ;; *) echo '--bin-dir must be absolute.' >&2; return 2;; esac
    asset=tailcat_${version#v}_${os}_${arch}.tar.gz
    base=https://github.com/LiuTangLei/tailcat-quic/releases/download/$version
    if [ "$dry_run" = true ]; then
        printf 'version=%s\nasset=%s\nurl=%s/%s\ndestination=%s/tailcat\n' "$version" "$asset" "$base" "$asset" "$bindir"
        return 0
    fi
    command -v tar >/dev/null || { echo 'tar is required.' >&2; return 2; }
    fetch() {
        if command -v curl >/dev/null; then
            curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 --retry 2 --connect-timeout 15 --max-time 300 "$1" -o "$2"
        elif command -v wget >/dev/null; then
            wget -q --https-only --timeout=60 -O "$2" "$1"
        else
            echo 'curl or wget is required.' >&2; return 2
        fi
    }
    hash_file() {
        if command -v sha256sum >/dev/null; then sha256sum "$1" | awk '{print $1}'
        elif command -v shasum >/dev/null; then shasum -a 256 "$1" | awk '{print $1}'
        elif command -v sha256 >/dev/null; then sha256 -q "$1"
        else echo 'A SHA-256 utility is required.' >&2; return 2; fi
    }
    tmp=$(mktemp -d "${TMPDIR:-/tmp}/tailcat-install.XXXXXXXX")
    staged=
    trap 'rm -rf -- "$tmp"; if [ -n "$staged" ]; then rm -f -- "$staged"; fi' EXIT HUP INT TERM
    fetch "$base/checksums.txt" "$tmp/checksums.txt"
    expected=$(awk -v name="$asset" '$2==name {print $1}' "$tmp/checksums.txt")
    printf '%s\n' "$expected" | LC_ALL=C grep -Eq '^[0-9a-f]{64}$' || {
        echo 'Release checksum is missing or ambiguous.' >&2; return 1;
    }
    fetch "$base/$asset" "$tmp/archive.tar.gz"
    actual=$(hash_file "$tmp/archive.tar.gz")
    [ "$actual" = "$expected" ] || { echo 'SHA-256 mismatch; nothing installed.' >&2; return 1; }
    # Extract only the named executable to a fresh file; never extract archive paths.
    [ "$(tar -tzf "$tmp/archive.tar.gz" | grep -cx tailcat)" = 1 ] || {
        echo 'Invalid archive executable layout.' >&2; return 1;
    }
    tar -xOzf "$tmp/archive.tar.gz" tailcat > "$tmp/tailcat"
    chmod 755 "$tmp/tailcat"
    actual_version=$("$tmp/tailcat" version)
    [ "$actual_version" = "$version" ] || { echo 'Executable version mismatch; nothing installed.' >&2; return 1; }
    mkdir -p "$bindir"
    staged=$(mktemp "$bindir/.tailcat-install.XXXXXXXX")
    cp "$tmp/tailcat" "$staged"
    chmod 755 "$staged"
    mv -f "$staged" "$bindir/tailcat"
    staged=
    printf 'Installed %s at %s/tailcat (SHA-256 verified).\n' "$version" "$bindir"
    case ":${PATH:-}:" in *":$bindir:"*) ;; *) printf 'Add %s to PATH to run tailcat by name.\n' "$bindir";; esac
    echo 'No service was started and no existing identity/configuration was changed.'
}

tailcat_install "$@"

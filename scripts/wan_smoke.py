#!/usr/bin/env python3
"""Isolated CLI WAN integrity test. No production services or firewall changes.

The SSH host is supplied at runtime and is never written to the public report.
All remote processes have a hard timeout and are cleaned up by exact test PID.
"""
import argparse
import concurrent.futures
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import selectors
import shlex
import socket
import subprocess
import tempfile
import time

BLOCK = bytes(range(256)) * 4096


def read_line(proc, timeout=90):
    with selectors.DefaultSelector() as sel:
        sel.register(proc.stdout, selectors.EVENT_READ)
        if not sel.select(timeout):
            raise TimeoutError("test process did not report readiness")
    line = proc.stdout.readline()
    if not line:
        raise RuntimeError("test process exited before readiness")
    return line.decode().strip()


def expected_digest(size):
    h = hashlib.sha256()
    while size:
        data = BLOCK[:min(size, len(BLOCK))]
        h.update(data)
        size -= len(data)
    return h.hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--remote", required=True, help="SSH destination; kept out of reports")
    parser.add_argument("--local-binary", required=True)
    parser.add_argument("--linux-binary", required=True)
    parser.add_argument("--size-mib", type=int, default=32)
    parser.add_argument("--rounds", type=int, default=2)
    parser.add_argument("--relay", action="store_true")
    parser.add_argument("--remote-root", choices=["/tmp", "/dev/shm"], default="/tmp")
    args = parser.parse_args()
    if not 1 <= args.size_mib <= 128 or not 1 <= args.rounds <= 5:
        parser.error("size must be 1..128 MiB and rounds 1..5")
    ssh = ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", args.remote]
    run = lambda cmd: subprocess.run(ssh + [cmd], check=True, capture_output=True, text=True, timeout=30)
    remote_dir = run("mktemp -d " + args.remote_root + "/tailcat-h3-wan.XXXXXXXX").stdout.strip()
    if not re.fullmatch(re.escape(args.remote_root) + r"/tailcat-h3-wan\.[A-Za-z0-9]+", remote_dir):
        raise RuntimeError("unexpected temporary directory")
    children = []
    remote_pids = []
    logs = []
    try:
        subprocess.run(["scp", "-q", "-o", "BatchMode=yes", args.linux_binary,
                        f"{args.remote}:{remote_dir}/tailcat"], check=True, timeout=40)
        subprocess.run(["scp", "-q", "-o", "BatchMode=yes", str(Path(__file__).with_name("wan_service.py")),
                        f"{args.remote}:{remote_dir}/service.py"], check=True, timeout=40)
        run(f"chmod 700 {shlex.quote(remote_dir + '/tailcat')}")
        with tempfile.TemporaryDirectory(prefix="tailcat-h3-wan-") as private:
            def spawn_remote(command, name):
                log = open(Path(private) / (name + ".log"), "w+b")
                logs.append(log)
                wrapped = "printf '%s\\n' \"$$\"; exec timeout 300 " + command
                proc = subprocess.Popen(ssh + [wrapped], stdout=subprocess.PIPE, stderr=log, bufsize=0)
                children.append(proc)
                pid = read_line(proc, 20)
                if not pid.isdigit():
                    raise RuntimeError("missing remote supervisor PID")
                remote_pids.append(int(pid))
                return proc, log

            fixture, _ = spawn_remote("python3 -u " + shlex.quote(remote_dir + "/service.py"), "fixture")
            target_port = json.loads(read_line(fixture))["port"]
            relay_env = "env TMPDIR=" + shlex.quote(remote_dir) + " XDG_CACHE_HOME=" + shlex.quote(remote_dir + "/cache") + " "
            if args.relay:
                relay_env += "TS_DEBUG_ALWAYS_USE_DERP=true "
            server, server_log = spawn_remote(
                relay_env + shlex.quote(remote_dir + "/tailcat") +
                f" --key=new --verbose --json serve {target_port}", "server")
            code = json.loads(read_line(server))["listenAddr"]
            if not code.startswith("tch3"):
                raise RuntimeError("server emitted an incompatible connection code")
            with socket.socket() as probe:
                probe.bind(("127.0.0.1", 0))
                local_port = probe.getsockname()[1]
            client_log = open(Path(private) / "client.log", "w+b")
            logs.append(client_log)
            env = os.environ.copy()
            if args.relay:
                env["TS_DEBUG_ALWAYS_USE_DERP"] = "true"
            client = subprocess.Popen([args.local_binary, "--key=new", "--verbose", "forward", code,
                                       f"{local_port}:{target_port}"], stdout=subprocess.DEVNULL,
                                      stderr=client_log, env=env)
            children.append(client)
            deadline = time.monotonic() + 60
            while True:
                if client.poll() is not None:
                    raise RuntimeError("forward process exited")
                try:
                    with socket.create_connection(("127.0.0.1", local_port), timeout=1):
                        break
                except OSError:
                    if time.monotonic() > deadline:
                        raise TimeoutError("forward listener unavailable")
                    time.sleep(0.1)

            def transfer(direction, size):
                started = time.monotonic()
                conn = http.client.HTTPConnection("127.0.0.1", local_port, timeout=120)
                try:
                    if direction == "download":
                        conn.request("GET", f"/bytes/{size}")
                        resp = conn.getresponse()
                        h = hashlib.sha256()
                        count = 0
                        while data := resp.read(1024 * 1024):
                            h.update(data)
                            count += len(data)
                        good = resp.status == 200 and count == size and h.hexdigest() == expected_digest(size)
                    else:
                        conn.putrequest("POST", "/upload")
                        conn.putheader("Content-Length", str(size))
                        conn.endheaders()
                        remaining = size
                        while remaining:
                            data = BLOCK[:min(remaining, len(BLOCK))]
                            conn.send(data)
                            remaining -= len(data)
                        resp = conn.getresponse()
                        body = json.loads(resp.read())
                        good = resp.status == 200 and body == {"bytes": size, "sha256": expected_digest(size)}
                    if not good:
                        raise RuntimeError(direction + " integrity verification failed")
                    elapsed = time.monotonic() - started
                    return {"direction": direction, "bytes": size, "sha256_verified": True,
                            "seconds": round(elapsed, 3), "mbps": round(size * 8 / elapsed / 1e6, 2)}
                finally:
                    conn.close()

            # Warm-up establishes the authenticated H3 data path, not just discovery.
            results = [transfer("download", 1024)]
            for _ in range(args.rounds):
                results.append(transfer("upload", args.size_mib << 20))
                results.append(transfer("download", args.size_mib << 20))
            with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
                futures = [pool.submit(transfer, direction, 2 << 20)
                           for direction in ["upload", "download", "upload", "download"]]
                results.extend(f.result() for f in futures)
            time.sleep(6)
            results.append(transfer("download", 1 << 20))
            results.append(transfer("upload", 1 << 20))
            client_log.flush(); client_log.seek(0)
            server_log.flush(); server_log.seek(0)
            client_text = client_log.read().decode(errors="replace")
            server_text = server_log.read().decode(errors="replace")
            for text in (client_text, server_text):
                if "Creating native QUIC IP engine (no WireGuard device)" not in text:
                    raise RuntimeError("missing H3-only engine evidence")
                if args.relay and "TS_DEBUG_ALWAYS_USE_DERP" not in text:
                    raise RuntimeError("forced-relay setting was not exercised")
            direct_observed = "now using " in client_text and "now using " in server_text
            if not args.relay and not direct_observed:
                raise RuntimeError("data passed but direct path was not established on both endpoints")
            report = {"path": "forced-derp" if args.relay else "direct-udp", "h3_only": True,
                      "direct_observed": direct_observed, "results": results,
                      "total_verified_bytes": sum(r["bytes"] for r in results)}
            print(json.dumps(report, indent=2), flush=True)
    finally:
        for proc in reversed(children):
            if proc.poll() is None:
                proc.terminate()
        # Kill only supervisors whose current command still contains this unique directory.
        for pid in remote_pids:
            try:
                command = run(f"ps -p {pid} -o args=").stdout
                if remote_dir in command:
                    run(f"kill -TERM {pid}")
            except (subprocess.SubprocessError, OSError):
                pass
        for proc in children:
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill(); proc.wait()
        for log in logs:
            log.close()
        run("rm -rf -- " + shlex.quote(remote_dir))


if __name__ == "__main__":
    main()

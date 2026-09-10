#!/usr/bin/env python3
"""Isolated two-host tailcat CLI performance/integrity test.

SSH destinations are runtime inputs, never included in reports. Both machines
use private /dev/shm directories; no routes, firewalls or services are changed.
Requires Python >=3.9 and iperf3 on the test machines. Run 'drive --help'.
"""
import argparse
import concurrent.futures
import hashlib
import http.client
import http.server
import ipaddress
import json
import os
from pathlib import Path
import re
import selectors
import shlex
import signal
import socket
import socketserver
import statistics
import struct
import subprocess
import sys
import tempfile
import threading
import time

BLOCK = bytes(range(256)) * 4096


def summary(samples):
    a = sorted(samples)
    if not a:
        return {"samples": 0}
    return {"samples": len(a), "min_ms": round(a[0], 2),
            "median_ms": round(statistics.median(a), 2),
            "p95_ms": round(a[min(len(a)-1, int(len(a)*.95))], 2),
            "max_ms": round(a[-1], 2)}


def recv_exact(s, n):
    out = b""
    while len(out) < n:
        piece = s.recv(n-len(out))
        if not piece:
            raise RuntimeError("unexpected EOF in test protocol")
        out += piece
    return out


def readiness(p, deadline=60):
    with selectors.DefaultSelector() as sel:
        sel.register(p.stdout, selectors.EVENT_READ)
        if not sel.select(deadline):
            raise TimeoutError("test server did not announce readiness")
    raw = p.stdout.readline()
    if not raw:
        raise RuntimeError("test server exited before readiness")
    return json.loads(raw)


def stop(p):
    if p and p.poll() is None:
        os.killpg(p.pid, signal.SIGTERM)
        try:
            p.wait(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(p.pid, signal.SIGKILL)
            p.wait()


def env_for(path, relay):
    env = os.environ.copy()
    env.update(TMPDIR=str(path), XDG_CACHE_HOME=str(path / "cache"))
    env.pop("TS_DEBUG_ALWAYS_USE_DERP", None)
    if relay:
        env["TS_DEBUG_ALWAYS_USE_DERP"] = "true"
    return env


def spawn(command, path, name, relay):
    log_path = path / (name + ".log")
    log = open(log_path, "wb")
    p = subprocess.Popen(command, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                         stderr=log, env=env_for(path, relay), start_new_session=True)
    log.close()
    return p, log_path


class ProcessUsage:
    def __init__(self, pid):
        self.pid = pid
        self.done = threading.Event()
        self.cpu, self.rss = [], []
        self.thread = threading.Thread(target=self.sample, daemon=True)
        self.thread.start()

    def sample(self):
        previous = None
        while not self.done.is_set():
            try:
                fields = Path(f"/proc/{self.pid}/stat").read_text().split(")", 1)[1].split()
                used = (int(fields[11]) + int(fields[12])) / os.sysconf("SC_CLK_TCK")
                now = time.monotonic()
                if previous:
                    self.cpu.append(100*(used-previous[0])/(now-previous[1]))
                previous = (used, now)
                status = Path(f"/proc/{self.pid}/status").read_text()
                match = re.search(r"VmRSS:\s+(\d+) kB", status)
                if match:
                    self.rss.append(int(match[1])/1024)
            except (OSError, ValueError, IndexError):
                break
            self.done.wait(.5)

    def result(self):
        self.done.set()
        self.thread.join(timeout=2)
        return {"cpu_peak_percent_one_core": round(max(self.cpu, default=0), 1),
                "cpu_mean_percent_one_core": round(statistics.mean(self.cpu), 1) if self.cpu else 0,
                "peak_rss_mib": round(max(self.rss, default=0), 1)}


def log_evidence(path):
    text = path.read_text(errors="replace")
    metrics = []
    for line in text.splitlines():
        if "tailcat-perf " in line:
            try:
                metrics.append(json.loads(line.split("tailcat-perf ", 1)[1]))
            except json.JSONDecodeError:
                pass
    return {"perf_samples": metrics,"native_h3_no_wg": "Creating native QUIC IP engine (no WireGuard device)" in text,
            "native_wg": "Creating WireGuard device" in text,
            "direct_seen": "now using " in text,
            "relay_forced": "TS_DEBUG_ALWAYS_USE_DERP" in text,
            "panic_seen": "panic:" in text}


class TCPServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True


class Echo(socketserver.BaseRequestHandler):
    def handle(self):
        self.request.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        while data := self.request.recv(65536):
            self.request.sendall(data)


class UDPServer(socketserver.ThreadingUDPServer):
    daemon_threads = True


class UDPEcho(socketserver.BaseRequestHandler):
    def handle(self):
        data, s = self.request
        s.sendto(data, self.client_address)


class HTTP(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *_):
        pass

    def do_GET(self):
        try:
            size = int(self.path.removeprefix("/bytes/"))
            if not 0 <= size <= 128 << 20:
                raise ValueError()
        except ValueError:
            self.send_error(400)
            return
        self.send_response(200)
        self.send_header("Content-Length", str(size))
        self.end_headers()
        while size:
            data = BLOCK[:min(len(BLOCK), size)]
            self.wfile.write(data)
            size -= len(data)

    def do_POST(self):
        size = int(self.headers.get("Content-Length", "-1"))
        if not 0 <= size <= 128 << 20:
            self.send_error(400)
            return
        h, left = hashlib.sha256(), size
        while left:
            data = self.rfile.read(min(left, len(BLOCK)))
            if not data:
                return
            h.update(data)
            left -= len(data)
        data = json.dumps({"bytes": size, "sha256": h.hexdigest()}).encode()
        self.send_response(200)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def server(args):
    path = Path(args.cli).resolve().parent
    children, services = [], []
    try:
        for cls, handler in [(TCPServer, Echo), (http.server.ThreadingHTTPServer, HTTP)]:
            svc = cls(("127.0.0.1", 0), handler)
            services.append(svc)
            threading.Thread(target=svc.serve_forever, daemon=True).start()
        echo_port, http_port = [s.server_address[1] for s in services]
        udp = UDPServer(("127.0.0.1", echo_port), UDPEcho)
        services.append(udp)
        threading.Thread(target=udp.serve_forever, daemon=True).start()
        iperf_port = free_port()
        ip, _ = spawn(["iperf3", "-s", "-B", "127.0.0.1", "-p", str(iperf_port)], path, "iperf", args.relay)
        children.append(ip)
        command = [args.cli, "--key=new", "--verbose", "--json", "serve", f"{echo_port},{http_port},{iperf_port}"]
        tc, log = spawn(command, path, "server", args.relay)
        children.append(tc)
        usage = ProcessUsage(tc.pid)
        info = readiness(tc)
        code = info["listenAddr"]
        if not code.startswith("tch3" if args.transport == "h3" else "tc"):
            raise RuntimeError("unexpected connection code")
        if args.transport == "wg" and code.startswith("tch3"):
            raise RuntimeError("WG baseline must use the official executable")
        udp_info = None
        if args.udp_driver:
            driver, _ = spawn([args.udp_driver, "-mode=server", "-timeout=290s"], path, "udp-api", args.relay)
            children.append(driver)
            udp_info = readiness(driver)
        # This line is consumed privately by the controlling SSH process.
        print(json.dumps({"code": code, "echo_port": echo_port, "udp_info": udp_info,
                          "http_port": http_port, "iperf_port": iperf_port}), flush=True)
        sys.stdin.readline()
        print(json.dumps({"server_evidence": log_evidence(log), "server_process": usage.result()}), flush=True)
    finally:
        for p in reversed(children):
            stop(p)
        for s in services:
            s.shutdown()
            s.server_close()


def wait_forward(p, log, ports):
    end = time.monotonic() + 60
    while time.monotonic() < end:
        text = log.read_text(errors="replace")
        found = {int(remote): int(local) for local, remote in
                 re.findall(r"forwarding 127\.0\.0\.1:(\d+) -> remote (?:localhost:)?(\d+)", text)}
        if all(x in found for x in ports):
            return found
        if p.poll() is not None:
            raise RuntimeError("tailcat forwarding process exited")
        time.sleep(.1)
    raise TimeoutError("local forwarding listeners not ready")


def echo_samples(port, count=20, finish=None):
    values, errors = [], 0
    try:
        with socket.create_connection(("127.0.0.1", port), timeout=30) as s:
            s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
            s.settimeout(5)
            i = 0
            while (finish is not None and not finish.is_set()) or (finish is None and i < count):
                payload = struct.pack("!Q", i) + bytes(range(56))
                started = time.perf_counter()
                s.sendall(payload)
                if recv_exact(s, len(payload)) != payload:
                    raise RuntimeError("echo bytes changed")
                values.append((time.perf_counter()-started)*1000)
                i += 1
                time.sleep(.10)
    except (OSError, RuntimeError):
        errors += 1
    result = summary(values)
    result["errors"] = errors
    return result


def hash_for(size):
    h = hashlib.sha256()
    while size:
        data = BLOCK[:min(size, len(BLOCK))]
        h.update(data)
        size -= len(data)
    return h.hexdigest()


def transfer(port, size, upload):
    started = time.perf_counter()
    conn = http.client.HTTPConnection("127.0.0.1", port, timeout=60)
    try:
        if upload:
            conn.putrequest("POST", "/upload")
            conn.putheader("Content-Length", str(size))
            conn.endheaders()
            left = size
            while left:
                data = BLOCK[:min(left, len(BLOCK))]
                conn.send(data)
                left -= len(data)
            resp = conn.getresponse()
            value = json.loads(resp.read())
            good = resp.status == 200 and value == {"bytes": size, "sha256": hash_for(size)}
        else:
            conn.request("GET", f"/bytes/{size}")
            resp = conn.getresponse()
            h, count = hashlib.sha256(), 0
            while data := resp.read(1 << 20):
                h.update(data)
                count += len(data)
            good = resp.status == 200 and count == size and h.hexdigest() == hash_for(size)
        elapsed = time.perf_counter()-started
        if not good:
            raise RuntimeError("application SHA-256 mismatch")
        return {"direction": "client-to-server" if upload else "server-to-client", "bytes": size,
                "seconds": round(elapsed, 3), "mbps": round(size*8/elapsed/1e6, 2), "sha256_ok": True}
    finally:
        conn.close()


def udp_probe(args, info, path):
    if args.udp_driver:
        command = [args.udp_driver, "-mode=client", "-timeout=90s",
                   "-expect-direct=" + ("false" if args.relay else "true")]
        out = subprocess.run(command, input=json.dumps(info["udp_info"]),
                             capture_output=True, text=True, timeout=100,
                             env=env_for(path, args.relay))
        if out.returncode:
            raise RuntimeError("UDP API probe failed: " + out.stderr[-400:])
        return json.loads(out.stdout)
    # Optional SOCKS check requires a CLI with explicit UDP service callbacks.
    # Numeric serve in the v0.6 baseline does not provide those callbacks.
    port = free_port()
    tc, log = spawn([args.cli, "--key=new", "--verbose", "socks", f"--listen=127.0.0.1:{port}", info["code"]],
                    path, "socks", args.relay)
    ctrl = None
    try:
        deadline = time.monotonic()+45
        while True:
            try:
                ctrl = socket.create_connection(("127.0.0.1", port), timeout=3)
                break
            except OSError:
                if tc.poll() is not None or time.monotonic() > deadline:
                    raise TimeoutError("SOCKS listener startup")
                time.sleep(.1)
        ctrl.settimeout(30)
        ctrl.sendall(b"\x05\x01\x00")
        if recv_exact(ctrl, 2) != b"\x05\x00":
            raise RuntimeError("SOCKS greeting failed")
        ctrl.sendall(b"\x05\x03\x00\x01" + b"\x00"*6)
        head = recv_exact(ctrl, 4)
        if head[:2] != b"\x05\x00":
            raise RuntimeError("SOCKS UDP associate failed")
        if head[3] == 1:
            host = socket.inet_ntop(socket.AF_INET, recv_exact(ctrl, 4))
        elif head[3] == 4:
            host = socket.inet_ntop(socket.AF_INET6, recv_exact(ctrl, 16))
        else:
            raise RuntimeError("unsupported SOCKS bind address")
        udpport = struct.unpack("!H", recv_exact(ctrl, 2))[0]
        name = b"server.tailcat"
        header = b"\x00\x00\x00\x03" + bytes([len(name)]) + name + struct.pack("!H", info["echo_port"])
        samples, lost = [], 0
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
            s.settimeout(15)
            # Cold UDP is unreliable: allow bounded warm-up retransmission,
            # count only subsequent probes in the loss measurement.
            warm = header + b"warmup"
            for _ in range(3):
                s.sendto(warm, (host, udpport))
                try:
                    data, _ = s.recvfrom(65535)
                    if data.endswith(b"warmup"):
                        break
                except socket.timeout:
                    continue
            else:
                raise TimeoutError("H3 UDP warm-up")
            for i in range(20):
                payload = struct.pack("!I", i) + BLOCK[:1196]
                started = time.perf_counter()
                s.settimeout(2)
                s.sendto(header+payload, (host, udpport))
                try:
                    data, _ = s.recvfrom(65535)
                    if not data.endswith(payload):
                        raise RuntimeError("UDP echo bytes changed")
                    samples.append((time.perf_counter()-started)*1000)
                except socket.timeout:
                    lost += 1
            return {"sent": 20, "received": len(samples), "lost": lost, "payload_bytes": 1200,
                    "rtt": summary(samples), "evidence": log_evidence(log)}
    finally:
        if ctrl:
            ctrl.close()
        stop(tc)


def client(args):
    info = json.loads(sys.stdin.readline())
    path = Path(args.cli).resolve().parent
    ports = [info[k] for k in ("echo_port", "http_port", "iperf_port")]
    command = [args.cli, "--key=new", "--verbose", "forward", info["code"]] + [f"0:{p}" for p in ports]
    tc, log = spawn(command, path, "client", args.relay)
    usage = ProcessUsage(tc.pid)
    result = {"path": "forced-derp" if args.relay else "direct-udp", "iperf": [], "integrity": []}
    try:
        mapping = wait_forward(tc, log, ports)
        ep, hp, ip = [mapping[x] for x in ports]
        start = time.monotonic()
        result["integrity"].append(transfer(hp, 1 << 10, False))
        result["cold_ready_seconds"] = round(time.monotonic()-start, 3)
        result["idle_rtt"] = echo_samples(ep)
        result["integrity"].extend(transfer(hp, args.size_mib << 20, up) for up in (True, False))
        for streams in ([1] if args.relay else [1, 4]):
            for reverse in (False, True):
                finished = threading.Event()
                latency = {}
                thread = threading.Thread(target=lambda: latency.update(echo_samples(ep, finish=finished)))
                thread.start()
                cmd = ["iperf3", "-c", "127.0.0.1", "-p", str(ip), "-t", str(args.seconds),
                       "-P", str(streams), "-O", "2", "-J", "--connect-timeout", "15000"]
                if reverse:
                    cmd.append("-R")
                started = time.monotonic()
                try:
                    out = subprocess.run(cmd, capture_output=True, text=True, timeout=args.seconds+45)
                    data = json.loads(out.stdout)
                    if out.returncode or "error" in data:
                        raise RuntimeError("iperf3 tunnel run failed: " + str(data.get("error", "exit status")))
                finally:
                    finished.set()
                    thread.join(timeout=8)
                end = data["end"]
                result["iperf"].append({"direction": "server-to-client" if reverse else "client-to-server",
                    "streams": streams, "seconds": round(time.monotonic()-started, 2),
                    "receiver_mbps": round(end["sum_received"]["bits_per_second"]/1e6, 2),
                    "sender_mbps": round(end["sum_sent"]["bits_per_second"]/1e6, 2),
                    "inner_tcp_retransmits": end["sum_sent"].get("retransmits"), "loaded_rtt": latency})
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
            size = (256 << 10) if args.relay else (2 << 20)
            futures = [pool.submit(transfer, hp, size, up) for up in (True, False, True, False)]
            result["concurrent_integrity"] = [f.result() for f in futures]
        time.sleep(6)
        result["idle_recovery"] = [transfer(hp, 256 << 10, up) for up in (True, False)]
        result["udp"] = udp_probe(args, info, path) if args.udp_driver else {"ok": True, "tested": False}
        result["client_evidence"] = log_evidence(log)
        result["total_sha256_bytes"] = sum(x["bytes"] for k in ("integrity", "concurrent_integrity", "idle_recovery") for x in result[k])
        if (args.transport == "h3" and not result["client_evidence"]["native_h3_no_wg"]) or result["client_evidence"]["panic_seen"]:
            raise RuntimeError("missing required engine evidence or panic observed")
        if not args.relay and not result["client_evidence"]["direct_seen"]:
            raise RuntimeError("data succeeded but no direct path was observed")
        result["ok"] = result["udp"].get("ok", result["udp"].get("lost", 0) == 0)
    except Exception as e:
        result["ok"], result["error"] = False, type(e).__name__ + ": " + str(e).replace(info["code"], "[credential]")
        result["client_evidence"] = log_evidence(log)
    finally:
        result["client_process"] = usage.result()
        stop(tc)
    print(json.dumps(result), flush=True)


def drive(args):
    dirs, procs = {}, []
    cleanup_errors = []
    hosts = [args.server, args.client]
    script = Path(__file__).resolve()
    binary = Path(args.binary).resolve()
    def ssh(host, cmd):
        return ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", host, cmd]
    try:
        for host in hosts:
            p = subprocess.run(ssh(host, "mktemp -d /dev/shm/tailcat-pair.XXXXXXXX"), check=True,
                               capture_output=True, text=True, timeout=20)
            path = p.stdout.strip()
            if not re.fullmatch(r"/dev/shm/tailcat-pair\.[A-Za-z0-9]+", path):
                raise RuntimeError("unsafe test directory")
            dirs[host] = path
            sources = [str(script), str(binary)]
            if args.udp_driver:
                sources.append(str(Path(args.udp_driver).resolve()))
            subprocess.run(["scp", "-C", "-q", "-o", "BatchMode=yes"] + sources +
                           [host+":"+path+"/"], check=True, timeout=120)
            subprocess.run(ssh(host, "chmod 700 " + shlex.quote(path+"/"+binary.name)), check=True, timeout=15)
        checksums = []
        for host in hosts:
            out = subprocess.run(ssh(host, "sha256sum " + shlex.quote(dirs[host]+"/"+binary.name)),
                                 check=True, capture_output=True, text=True, timeout=15)
            checksums.append(out.stdout.split()[0])
        local_digest = hashlib.sha256(binary.read_bytes()).hexdigest()
        if checksums != [local_digest, local_digest]:
            raise RuntimeError("remote executable checksum mismatch")
        relay = " --relay" if args.relay else ""
        def command(host, mode):
            path = dirs[host]
            driver_arg = ""
            if args.udp_driver:
                driver_arg = " --udp-driver " + shlex.quote(path+"/"+Path(args.udp_driver).name)
            return ("timeout 300 python3 " + shlex.quote(path+"/"+script.name) + " " + mode +
                    " --cli " + shlex.quote(path+"/"+binary.name) + relay + driver_arg +
                    f" --seconds {args.seconds} --size-mib {args.size_mib} --transport {args.transport}")
        p = subprocess.Popen(ssh(args.server, command(args.server, "server")), stdin=subprocess.PIPE,
                             stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True)
        procs.append(p)
        info = readiness(p, 90)
        result = subprocess.run(ssh(args.client, command(args.client, "client")),
                                input=json.dumps(info)+"\n", capture_output=True, text=True, timeout=280)
        if result.returncode:
            raise RuntimeError("remote client test process failed: " + result.stderr[-500:])
        report = json.loads(result.stdout)
        p.stdin.write(b"stop\n"); p.stdin.flush()
        report.update(readiness(p, 15))
        p.stdin.close()
        p.wait(timeout=20)
        report.update(binary_sha256=local_digest, server_label=args.server_label, client_label=args.client_label)
        if args.transport == "h3" and not report["server_evidence"]["native_h3_no_wg"]:
            report["ok"] = False
            report["server_error"] = "missing H3 engine evidence"
        report["transport"] = args.transport
        if args.report:
            destination = Path(args.report).resolve()
            if destination.exists():
                raise RuntimeError("refusing to overwrite an existing report")
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_text(json.dumps(report, indent=2)+"\n")
        printable = json.loads(json.dumps(report))
        for label in ("server_evidence", "client_evidence"):
            samples = printable.get(label, {}).pop("perf_samples", [])
            if samples:
                printable[label]["perf_final"] = samples[-1]
                printable[label]["perf_count"] = len(samples)
        print(json.dumps(printable, indent=2), flush=True)
    finally:
        for p in procs:
            stop(p)
        for host, path in dirs.items():
            # Match only this run's script/binary, not another tailcat process.
            cleanup = ("pids=$(pgrep -f '^"+re.escape(path+'/'+binary.name)+"( |$)' || true); "
                       "if [ -n \"$pids\" ]; then kill -TERM $pids 2>/dev/null || true; fi; "
                       "rm -rf -- " + shlex.quote(path))
            try:
                result = subprocess.run(ssh(host, cleanup), capture_output=True, text=True, timeout=30)
                if result.returncode:
                    cleanup_errors.append(f'{host}: {result.stderr[-500:]}')
            except Exception as exc:
                cleanup_errors.append(str(exc))
        if args.report and Path(args.report).is_file():
            destination = Path(args.report)
            saved = json.loads(destination.read_text())
            saved['cleanup_errors'] = cleanup_errors
            if cleanup_errors: saved['ok'] = False
            destination.write_text(json.dumps(saved, indent=2)+'\n')
        if cleanup_errors:
            raise RuntimeError('owned test cleanup failed: ' + '; '.join(cleanup_errors))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest="mode", required=True)
    for name in ("drive", "server", "client"):
        parser = sub.add_parser(name)
        parser.add_argument("--relay", action="store_true")
        parser.add_argument("--seconds", type=int, default=10)
        parser.add_argument("--transport", choices=["h3", "wg"], default="h3")
        parser.add_argument("--size-mib", type=int, default=12)
        parser.add_argument("--udp-driver", default="", help="separately compiled real UDP API probe")
        if name == "drive":
            parser.add_argument("--server", required=True)
            parser.add_argument("--client", required=True)
            parser.add_argument("--binary", required=True)
            parser.add_argument("--report", default="", help="new local JSON test evidence file")
            parser.add_argument("--server-label", default="server")
            parser.add_argument("--client-label", default="client")
        else:
            parser.add_argument("--cli", required=True)
    args = p.parse_args()
    if not 3 <= args.seconds <= 30 or not 1 <= args.size_mib <= 64:
        p.error("seconds must be 3..30; size-mib must be 1..64")
    if args.mode != "drive":
        def terminated(*_):
            raise KeyboardInterrupt()
        signal.signal(signal.SIGTERM, terminated)
    globals()[args.mode](args)


if __name__ == "__main__":
    main()

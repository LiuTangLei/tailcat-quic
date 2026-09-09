#!/usr/bin/env python3
"""Bounded native TCP/UDP baseline between two explicitly supplied SSH hosts.

Only a transient iperf3 listener is created. No firewall, route, service or
congestion settings are changed. Raw addresses are omitted from the report.
"""
import argparse
import ipaddress
import json
import os
import shlex
import signal
import subprocess
import time


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--server", required=True)
    p.add_argument("--client", required=True)
    p.add_argument("--server-ip", required=True)
    p.add_argument("--port", type=int, default=39397)
    p.add_argument("--seconds", type=int, default=10)
    args = p.parse_args()
    ip = str(ipaddress.ip_address(args.server_ip))
    if not 1024 <= args.port <= 65535 or not 5 <= args.seconds <= 20:
        p.error("port must be 1024..65535; seconds must be 5..20")
    def ssh(host, command):
        return ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=8", host, command]
    report = []
    cases = [("tcp", 1, False), ("tcp", 1, True), ("tcp", 4, False),
             ("tcp", 4, True), ("udp-50m", 1, False), ("udp-50m", 1, True)]
    for protocol, streams, reverse in cases:
        server_cmd = ["env", "TMPDIR=/dev/shm", "timeout", "45", "iperf3", "-s", "-1", "-J", "-B", ip, "-p", str(args.port)]
        server = subprocess.Popen(ssh(args.server, shlex.join(server_cmd)), stdout=subprocess.PIPE,
                                  stderr=subprocess.PIPE, start_new_session=True)
        try:
            for _ in range(8):
                ready = subprocess.run(ssh(args.server, "ss -H -ltn 'sport = :"+str(args.port)+"'"),
                                       capture_output=True, text=True, timeout=10)
                if ready.stdout.strip():
                    break
                if server.poll() is not None:
                    stdout, stderr = server.communicate()
                    raise RuntimeError("native server startup: " + (stdout+stderr).decode(errors="replace")[-1200:])
                time.sleep(.3)
            else:
                raise TimeoutError("native baseline listener not ready")
            client_cmd = ["env", "TMPDIR=/dev/shm", "timeout", "35", "iperf3", "-c", ip, "-p", str(args.port),
                          "-t", str(args.seconds), "-O", "2", "-P", str(streams), "-J", "--connect-timeout", "5000"]
            if reverse:
                client_cmd.append("-R")
            if protocol.startswith("udp"):
                client_cmd += ["-u", "-b", "50M", "-l", "1200"]
            out = subprocess.run(ssh(args.client, shlex.join(client_cmd)), capture_output=True, text=True, timeout=40)
            data = json.loads(out.stdout)
            if out.returncode or "error" in data:
                raise RuntimeError("native test failed: " + data.get("error", "process exit"))
            end = data["end"]
            row = {"protocol": protocol, "streams": streams,
                   "direction": "server-to-client" if reverse else "client-to-server"}
            if protocol == "tcp":
                row.update(receiver_mbps=round(end["sum_received"]["bits_per_second"]/1e6, 2),
                           sender_mbps=round(end["sum_sent"]["bits_per_second"]/1e6, 2),
                           retransmits=end["sum_sent"].get("retransmits"))
            else:
                summary = end.get("sum_received", end.get("sum", {}))
                row.update(receiver_mbps=round(summary["bits_per_second"]/1e6, 2),
                           lost_percent=summary.get("lost_percent"), jitter_ms=summary.get("jitter_ms"),
                           packets=summary.get("packets"), lost_packets=summary.get("lost_packets"))
            report.append(row)
            server.communicate(timeout=8)
        finally:
            if server.poll() is None:
                os.killpg(server.pid, signal.SIGTERM)
                try:
                    server.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    os.killpg(server.pid, signal.SIGKILL)
                    server.wait()
            time.sleep(.5)
    print(json.dumps({"kind": "native-link-baseline", "results": report}, indent=2))


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Loopback-only deterministic HTTP fixture for the isolated WAN smoke test."""
import hashlib
import http.server
import json
import socketserver
import sys

BLOCK = bytes(range(256)) * 4096
LIMIT = 256 * 1024 * 1024

class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *_):
        pass

    def do_GET(self):
        try:
            size = int(self.path.removeprefix("/bytes/"))
            if not 0 <= size <= LIMIT:
                raise ValueError()
        except ValueError:
            self.send_error(400)
            return
        self.send_response(200)
        self.send_header("Content-Length", str(size))
        self.end_headers()
        while size:
            chunk = BLOCK[:min(size, len(BLOCK))]
            self.wfile.write(chunk)
            size -= len(chunk)

    def do_POST(self):
        try:
            size = int(self.headers.get("Content-Length", "-1"))
            if not 0 <= size <= LIMIT:
                raise ValueError()
        except ValueError:
            self.send_error(400)
            return
        digest = hashlib.sha256()
        remaining = size
        while remaining:
            data = self.rfile.read(min(remaining, 1024 * 1024))
            if not data:
                return
            digest.update(data)
            remaining -= len(data)
        body = json.dumps({"bytes": size, "sha256": digest.hexdigest()}).encode()
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

class Server(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True

with Server(("127.0.0.1", 0), Handler) as server:
    print(json.dumps({"port": server.server_port}), flush=True)
    server.serve_forever()

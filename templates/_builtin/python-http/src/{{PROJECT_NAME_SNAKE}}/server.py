"""A small HTTP service using only the standard library."""

import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any


class Handler(BaseHTTPRequestHandler):
    def _send(self, status: int, body: Any) -> None:
        payload = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self) -> None:  # noqa: N802 (the name is fixed by http.server)
        path = self.path.split("?", 1)[0]
        if path == "/healthz":
            self._send(200, {"status": "ok"})
        elif path == "/":
            self._send(200, {"message": "Hello from {{PROJECT_NAME}}"})
        else:
            self._send(404, {"error": "not found"})

    def log_message(self, format: str, *args: Any) -> None:  # keep tests quiet
        if os.environ.get("{{PROJECT_NAME_UPPER}}_LOG"):
            super().log_message(format, *args)


def make_server(host: str = "127.0.0.1", port: int = 8080) -> ThreadingHTTPServer:
    return ThreadingHTTPServer((host, port), Handler)


def main() -> int:
    port = int(os.environ.get("PORT", "8080"))
    server = make_server("0.0.0.0", port)
    print(f"{{PROJECT_NAME}} listening on :{port}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0

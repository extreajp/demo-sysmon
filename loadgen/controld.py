#!/usr/bin/env python3
"""Loadgen control daemon on :8000."""

from __future__ import annotations

import json
import os
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from core import run_scenario

WEB_URL = os.environ.get("WEB_URL", "http://web:3000")
SYSMON_URL = os.environ.get("SYSMON_URL", "http://host.docker.internal:9101")
OUT = os.environ.get("OUT_PATH", "/out/correlation.csv")

_lock = threading.Lock()
_state = {
    "running": False,
    "scenario": "",
    "phase": "",
    "rows_written": 0,
    "error": "",
}
_stop = threading.Event()
_worker: threading.Thread | None = None


def _run(scenario: str) -> None:
    global _worker
    try:
        def on_phase(name: str) -> None:
            with _lock:
                _state["phase"] = name

        n = run_scenario(scenario, WEB_URL, SYSMON_URL, OUT, _stop, on_phase)
        with _lock:
            _state["rows_written"] = n
    except Exception as e:  # noqa: BLE001 — daemon reports any failure
        with _lock:
            _state["error"] = str(e)
    finally:
        with _lock:
            _state["running"] = False
            _state["phase"] = ""
        _worker = None


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args) -> None:
        return

    def _json(self, code: int, obj: dict) -> None:
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/health":
            self._json(200, {"ok": True})
            return
        if self.path == "/status":
            with _lock:
                self._json(200, dict(_state))
            return
        self._json(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        global _worker
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length else b"{}"
        try:
            body = json.loads(raw.decode() or "{}")
        except json.JSONDecodeError:
            body = {}
        if self.path == "/run":
            scenario = body.get("scenario") or "ramp"
            with _lock:
                if _state["running"]:
                    self._json(409, {"error": "already running", **_state})
                    return
                _stop.clear()
                _state.update(
                    {"running": True, "scenario": scenario, "phase": "", "rows_written": 0, "error": ""}
                )
            _worker = threading.Thread(target=_run, args=(scenario,), daemon=True)
            _worker.start()
            self._json(202, {"ok": True, "scenario": scenario})
            return
        if self.path == "/stop":
            _stop.set()
            with _lock:
                _state["running"] = False
            self._json(200, {"ok": True})
            return
        self._json(404, {"error": "not found"})


def main() -> None:
    os.makedirs(os.path.dirname(OUT) or ".", exist_ok=True)
    httpd = ThreadingHTTPServer(("0.0.0.0", 8000), Handler)
    httpd.serve_forever()


if __name__ == "__main__":
    main()

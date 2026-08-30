"""Scenarios, HTTP load, correlation CSV."""

from __future__ import annotations

import csv
import json
import threading
import time
import urllib.error
import urllib.request
from typing import Callable

SCENARIOS = {
    "ramp": [
        {"name": "warmup", "path": "/api/load/work?ms=30", "rps": 10, "duration_sec": 20},
        {"name": "work-ramp", "path": "/api/load/work?ms=50", "rps": 30, "duration_sec": 40},
        {"name": "cpu-ramp", "path": "/api/load/cpu?ms=500", "rps": 20, "duration_sec": 40},
        {"name": "cooldown", "path": "/api/load/work?ms=20", "rps": 5, "duration_sec": 20},
    ],
    "cpu-burst": [
        {"name": "warmup", "path": "/api/load/work?ms=30", "rps": 10, "duration_sec": 10},
        {"name": "burst", "path": "/api/load/cpu?ms=2000", "rps": 15, "duration_sec": 20},
        {"name": "cooldown", "path": "/api/load/work?ms=20", "rps": 5, "duration_sec": 10},
    ],
    "io-burst": [
        {"name": "warmup", "path": "/api/load/work?ms=30", "rps": 10, "duration_sec": 10},
        {"name": "burst", "path": "/api/load/io?mb=32", "rps": 8, "duration_sec": 20},
        {"name": "cooldown", "path": "/api/load/work?ms=20", "rps": 5, "duration_sec": 10},
    ],
}

CSV_FIELDS = [
    "ts",
    "phase",
    "p50_ms",
    "p99_ms",
    "rps",
    "error_rate",
    "host_psi_cpu",
    "cgroup_psi_cpu",
    "cpu_usage",
    "load1",
]


def _get(url: str, timeout: float = 5.0) -> dict:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as r:
            return json.loads(r.read().decode())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError):
        return {}


def _post(url: str, body: dict | None = None, timeout: float = 5.0) -> None:
    data = json.dumps(body or {}).encode()
    req = urllib.request.Request(url, data=data, method="POST", headers={"Content-Type": "application/json"})
    try:
        urllib.request.urlopen(req, timeout=timeout).read()
    except (urllib.error.URLError, TimeoutError, OSError):
        pass


def _sample_name(samples: list, name: str) -> float:
    for s in samples:
        if s.get("name") == name:
            return float(s.get("value") or 0)
    return 0.0


def fire(url: str, timeout: float) -> None:
    try:
        urllib.request.urlopen(url, timeout=timeout).read()
    except (urllib.error.URLError, TimeoutError, OSError):
        pass


def run_scenario(
    scenario: str,
    web_url: str,
    sysmon_url: str,
    out_path: str,
    stop_event: threading.Event,
    on_phase: Callable[[str], None] | None = None,
) -> int:
    phases = SCENARIOS.get(scenario)
    if not phases:
        raise ValueError("unknown scenario: " + scenario)

    rows = 0
    with open(out_path, "w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=CSV_FIELDS)
        w.writeheader()
        for phase in phases:
            if stop_event.is_set():
                break
            name = phase["name"]
            if on_phase:
                on_phase(name)
            _post(web_url.rstrip("/") + "/api/scenario", {"phase": name})
            target = web_url.rstrip("/") + phase["path"]
            rps = max(1, int(phase["rps"]))
            interval = 1.0 / rps
            end = time.time() + int(phase["duration_sec"])
            next_hit = time.time()
            next_row = time.time()
            while time.time() < end and not stop_event.is_set():
                now = time.time()
                if now >= next_hit:
                    threading.Thread(target=fire, args=(target, 8.0), daemon=True).start()
                    next_hit += interval
                if now >= next_row:
                    stats = _get(web_url.rstrip("/") + "/api/stats")
                    snap = _get(sysmon_url.rstrip("/") + "/api/snapshot")
                    samples = (snap.get("snapshot") or {}).get("samples") or []
                    lat = stats.get("latencyMs") or {}
                    w.writerow(
                        {
                            "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                            "phase": name,
                            "p50_ms": lat.get("p50", 0),
                            "p99_ms": lat.get("p99", 0),
                            "rps": stats.get("rps", 0),
                            "error_rate": stats.get("errorRate", 0),
                            "host_psi_cpu": _sample_name(samples, "host.psi.cpu.some.avg10"),
                            "cgroup_psi_cpu": _sample_name(samples, "cgroup.web.psi.cpu.some.avg10"),
                            "cpu_usage": _sample_name(samples, "cpu.usage_percent"),
                            "load1": _sample_name(samples, "loadavg.1"),
                        }
                    )
                    f.flush()
                    rows += 1
                    next_row += 1.0
                time.sleep(min(0.05, max(0.0, next_hit - time.time())))
    _post(web_url.rstrip("/") + "/api/scenario", {"phase": ""})
    return rows

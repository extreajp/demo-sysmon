#!/usr/bin/env python3
"""One-shot scenario runner."""

from __future__ import annotations

import argparse
import os
import sys
import threading

from core import run_scenario


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--scenario", default="ramp")
    p.add_argument("--web", default=os.environ.get("WEB_URL", "http://web:3000"))
    p.add_argument("--sysmon", default=os.environ.get("SYSMON_URL", "http://host.docker.internal:9101"))
    p.add_argument("--out", default="/out/correlation.csv")
    args = p.parse_args()
    stop = threading.Event()
    try:
        n = run_scenario(args.scenario, args.web, args.sysmon, args.out, stop)
    except ValueError as e:
        print(e, file=sys.stderr)
        return 2
    print(f"wrote {n} rows to {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

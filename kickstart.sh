#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

RUN_DIR="$ROOT/.run"
BIN="$ROOT/bin/sysmon"
CONFIG="$ROOT/config.example.json"
WEB_PORT="${WEB_PORT:-3000}"
SYSMON_PORT="${SYSMON_PORT:-9101}"
FORCE_BUILD=0

info() { printf '[kickstart] %s\n' "$*"; }
die() { printf '[kickstart] error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage: ./kickstart.sh [command] [options]
       ./kickstart.sh              # same as: up

Commands:
  up       Start sysmon and docker compose stack (default)
  down     Stop stack and sysmon
  status   Show PID, compose, snapshot excerpt
  bench    Run loadgen scenario (default: ramp)
  logs     Tail logs (sysmon|web|loadgen)
  purge    down + remove .run/, out/, bin/sysmon

Options:
  --build          Force compose rebuild
  --web-port P     Web port (default 3000)
  --sysmon-port P  sysmon port (default 9101)
EOF
}

need_linux() {
  [[ "$(uname -s)" == Linux ]] || die "Linux only"
}

need_docker() {
  command -v docker >/dev/null || die "docker not found (apt install docker.io docker-compose-v2)"
  docker compose version >/dev/null 2>&1 || die "docker compose v2 not found"
}

ensure_dirs() {
  mkdir -p "$RUN_DIR" "$ROOT/bin" "$ROOT/out"
}

sysmon_pid() {
  [[ -f "$RUN_DIR/sysmon.pid" ]] || return 1
  local pid
  pid="$(cat "$RUN_DIR/sysmon.pid")"
  kill -0 "$pid" 2>/dev/null || return 1
  printf '%s' "$pid"
}

stop_sysmon() {
  local pid n=0
  if pid="$(sysmon_pid)"; then
    if [[ -r /proc/$pid/cmdline ]] && grep -q -a -F "$BIN" /proc/$pid/cmdline; then
      kill "$pid" 2>/dev/null || true
      while kill -0 "$pid" 2>/dev/null && (( n < 20 )); do
        sleep 0.1
        n=$((n + 1))
      done
      kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
    fi
  fi
  rm -f "$RUN_DIR/sysmon.pid" "$RUN_DIR/sysmon.log"
}

ensure_sysmon_bin() {
  if [[ -x "$BIN" ]]; then
    info "Using existing $BIN"
    return
  fi
  if command -v go >/dev/null; then
    info "Building sysmon..."
    go build -o "$BIN" ./cmd/sysmon/
    return
  fi
  local arch tag url
  case "$(uname -m)" in
    x86_64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "unsupported arch $(uname -m); install golang-go and retry" ;;
  esac
  tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  [[ -n "$tag" ]] || die "go not found and no git tag for Releases; install golang-go"
  url="https://github.com/extreajp/demo-sysmon/releases/download/${tag}/sysmon-linux-${arch}"
  info "Fetching $url"
  curl -fsSL "$url" -o "$BIN"
  chmod +x "$BIN"
}

start_sysmon() {
  if sysmon_pid >/dev/null; then
    info "sysmon already running (pid $(sysmon_pid))"
    return
  fi
  ensure_sysmon_bin
  info "Starting sysmon on 127.0.0.1:${SYSMON_PORT}..."
  nohup "$BIN" serve \
    --config "$CONFIG" \
    --listen "127.0.0.1:${SYSMON_PORT}" \
    --cors-origin "http://127.0.0.1:${WEB_PORT}" \
    >"$RUN_DIR/sysmon.log" 2>&1 &
  echo $! >"$RUN_DIR/sysmon.pid"
  sleep 0.3
  if ! sysmon_pid >/dev/null; then
    die "sysmon exited; see $RUN_DIR/sysmon.log"
  fi
}

wait_http() {
  local url="$1" name="$2" n=0
  while (( n < 60 )); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      info "$name is ready"
      return 0
    fi
    n=$((n + 1))
    sleep 1
  done
  die "$name did not become ready: $url"
}

cmd_up() {
  need_linux
  need_docker
  ensure_dirs
  export WEB_PORT SYSMON_PORT
  local args=(up -d)
  (( FORCE_BUILD )) && args+=(--build)
  info "Starting docker compose stack..."
  docker compose "${args[@]}"
  start_sysmon
  wait_http "http://127.0.0.1:${SYSMON_PORT}/healthz" sysmon
  wait_http "http://127.0.0.1:${WEB_PORT}/api/health" web
  cat <<EOF

sysmon-demo is up.

  Dashboard:  http://127.0.0.1:${WEB_PORT}
  sysmon:     http://127.0.0.1:${SYSMON_PORT}/healthz

  Stop:  ./kickstart.sh down
  Logs:  ./kickstart.sh logs sysmon
EOF
}

cmd_down() {
  need_docker
  docker compose down --remove-orphans >/dev/null 2>&1 || true
  local ids
  ids="$(docker compose ps -aq 2>/dev/null || true)"
  [[ -n "$ids" ]] && docker rm -f $ids >/dev/null 2>&1 || true
  stop_sysmon
  info "stopped"
}

cmd_status() {
  local pid="-"
  if pid="$(sysmon_pid)"; then
    :
  else
    pid="-"
  fi
  printf 'sysmon pid: %s\n' "$pid"
  if command -v docker >/dev/null; then
    docker compose ps || true
  fi
  if curl -fsS "http://127.0.0.1:${SYSMON_PORT}/api/snapshot" >/tmp/sysmon-snap.json 2>/dev/null; then
    python3 - <<'PY'
import json
d=json.load(open("/tmp/sysmon-snap.json"))
print("firing:", d.get("firing"))
for s in d.get("snapshot", {}).get("samples", [])[:8]:
    print(f"  {s.get('name')}={s.get('value')}")
PY
  else
    echo "snapshot: unavailable"
  fi
}

cmd_bench() {
  need_docker
  local scenario="${1:-ramp}"
  mkdir -p "$ROOT/out"
  docker compose run --rm \
    -e WEB_URL=http://web:3000 \
    -e SYSMON_URL="http://host.docker.internal:${SYSMON_PORT}" \
    loadgen python3 bench.py --scenario "$scenario" --out /out/correlation.csv
}

cmd_logs() {
  local target="${1:-sysmon}"
  case "$target" in
    sysmon) tail -f "$RUN_DIR/sysmon.log" ;;
    web|loadgen) docker compose logs -f "$target" ;;
    *) die "logs target: sysmon|web|loadgen" ;;
  esac
}

cmd_purge() {
  cmd_down
  rm -rf "$RUN_DIR" "$ROOT/out" "$BIN"
  info "purged"
}

COMMAND=""
POSITIONAL=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --build) FORCE_BUILD=1; shift ;;
    --web-port) WEB_PORT="$2"; shift 2 ;;
    --sysmon-port) SYSMON_PORT="$2"; shift 2 ;;
    up|down|status|bench|logs|purge)
      COMMAND="$1"
      shift
      ;;
    *)
      if [[ -n "$COMMAND" ]]; then
        POSITIONAL+=("$1")
        shift
      else
        die "unknown argument: $1 (see --help)"
      fi
      ;;
  esac
done
if [[ -z "$COMMAND" ]]; then
  (( FORCE_BUILD )) && die "use: ./kickstart.sh up --build"
  COMMAND=up
fi
if (( FORCE_BUILD )) && [[ "$COMMAND" != up ]]; then
  die "--build is only valid with up (./kickstart.sh up --build)"
fi

case "$COMMAND" in
  up) cmd_up ;;
  down) cmd_down ;;
  status) cmd_status ;;
  bench) cmd_bench "${POSITIONAL[@]+"${POSITIONAL[@]}"}" ;;
  logs) cmd_logs "${POSITIONAL[@]+"${POSITIONAL[@]}"}" ;;
  purge) cmd_purge ;;
esac

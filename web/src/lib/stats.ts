const WINDOW_MS = 5000;
const RPS_MS = 1000;
const LAT_CAP = 400;

type Rec = { t: number; ms: number; err: boolean };
type Start = { t: number; cpuMs: number };
type Pending = { t0: number; cpuMs: number };

const recs: Rec[] = [];
const starts: Start[] = [];
const pending: Pending[] = [];
let inFlight = 0;
let totalRecv = 0;
let totalDone = 0;

function prune(now: number) {
  while (recs.length && now - recs[0].t > WINDOW_MS) recs.shift();
  while (recs.length > LAT_CAP) recs.shift();
  while (starts.length && now - starts[0].t > RPS_MS) starts.shift();
}

export function begin(cpuMs = 0): number {
  const now = Date.now();
  inFlight++;
  totalRecv++;
  starts.push({ t: now, cpuMs });
  pending.push({ t0: now, cpuMs });
  prune(now);
  return now;
}

export function end(t0: number, err = false) {
  inFlight = Math.max(0, inFlight - 1);
  const i = pending.findIndex((p) => p.t0 === t0);
  if (i >= 0) pending.splice(i, 1);
  const now = Date.now();
  recs.push({ t: now, ms: now - t0, err });
  totalDone++;
  prune(now);
}

function percentile(sorted: number[], p: number): number {
  if (!sorted.length) return 0;
  const i = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1));
  return sorted[i];
}

export function snapshot() {
  const now = Date.now();
  prune(now);
  const lats = recs.map((r) => r.ms).sort((a, b) => a - b);
  const errs = recs.filter((r) => r.err).length;
  const recv = starts.length;
  const done = recs.filter((r) => now - r.t <= RPS_MS).length;
  const offer = starts.reduce((sum, s) => sum + s.cpuMs, 0) / 1000;
  const backlog = pending.reduce((sum, p) => sum + p.cpuMs, 0) / 1000;
  if (inFlight === 0 && recv === 0 && done === 0) {
    totalRecv = 0;
    totalDone = 0;
  }
  return {
    latencyMs: {
      p50: percentile(lats, 50),
      p90: percentile(lats, 90),
      p99: percentile(lats, 99),
    },
    recv,
    done,
    wait: inFlight,
    totalRecv,
    totalDone,
    offer,
    backlog,
    rps: done,
    errorRate: recs.length ? errs / recs.length : 0,
    inFlight,
  };
}

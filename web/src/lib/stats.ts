const WINDOW_MS = 5000;
const LAT_CAP = 400;

type Rec = { t: number; ms: number; err: boolean };

const recs: Rec[] = [];
let inFlight = 0;

function prune(now: number) {
  while (recs.length && now - recs[0].t > WINDOW_MS) recs.shift();
  while (recs.length > LAT_CAP) recs.shift();
}

export function begin(): number {
  inFlight++;
  return Date.now();
}

export function end(t0: number, err = false) {
  inFlight = Math.max(0, inFlight - 1);
  const now = Date.now();
  recs.push({ t: now, ms: now - t0, err });
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
  const rps = recs.length / (WINDOW_MS / 1000);
  return {
    latencyMs: {
      p50: percentile(lats, 50),
      p90: percentile(lats, 90),
      p99: percentile(lats, 99),
    },
    rps,
    errorRate: recs.length ? errs / recs.length : 0,
    inFlight,
  };
}

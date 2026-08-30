"use client";

import { useEffect, useRef, useState } from "react";
import { Sparkline } from "./Sparkline";
import styles from "./Dashboard.module.css";

const SYSMON_URL = process.env.NEXT_PUBLIC_SYSMON_URL ?? "http://127.0.0.1:9101";
const CAP = 120;

type Sample = { name: string; value: number; labels?: Record<string, string> };
type Payload = {
  snapshot?: { samples?: Sample[] };
  alerts?: { name: string; firing: boolean }[];
  firing?: number;
};
type Bench = { running?: boolean; scenario?: string; phase?: string };

const CPU_CAP = 1;
const HINT: Record<string, string> = {
  ramp: "light 30–50ms work, then 500ms CPU",
  "cpu-burst": "2000ms × 2 workers × 15/s ≈ 60× cap",
  "io-burst": "32MB sync write, no CPU cap",
};

function push(buf: number[], v: number): number[] {
  const next = buf.length >= CAP ? buf.slice(buf.length - CAP + 1) : buf.slice();
  next.push(v);
  return next;
}

function val(samples: Sample[], name: string): number {
  return samples.find((s) => s.name === name)?.value ?? 0;
}

function writeSectors(samples: Sample[]): number {
  return samples
    .filter((s) => s.name === "disk.write_sectors")
    .reduce((sum, s) => sum + s.value, 0);
}

export function Dashboard() {
  const [sse, setSse] = useState<"connected" | "disconnected">("disconnected");
  const [cpu, setCpu] = useState(0);
  const [mem, setMem] = useState(0);
  const [load1, setLoad1] = useState(0);
  const [hostPsiCpu, setHostPsiCpu] = useState(0);
  const [hostPsiIo, setHostPsiIo] = useState(0);
  const [writeMbs, setWriteMbs] = useState(0);
  const [cgroupPsiCpu, setCgroupPsiCpu] = useState(0);
  const [cgroupPsiIo, setCgroupPsiIo] = useState(0);
  const [firing, setFiring] = useState(0);
  const [pressure, setPressure] = useState(false);
  const [hostHistCpu, setHostHistCpu] = useState<number[]>([]);
  const [hostHistIo, setHostHistIo] = useState<number[]>([]);
  const [hostHistWrite, setHostHistWrite] = useState<number[]>([]);
  const [cgHistCpu, setCgHistCpu] = useState<number[]>([]);
  const [cgHistIo, setCgHistIo] = useState<number[]>([]);
  const [p99Hist, setP99Hist] = useState<number[]>([]);
  const [stats, setStats] = useState({
    p50: 0, p99: 0, recv: 0, done: 0, wait: 0, totalRecv: 0, totalDone: 0,
    offer: 0, backlog: 0, errorRate: 0,
  });
  const [bench, setBench] = useState<Bench>({});
  const [scenario, setScenario] = useState("ramp");
  const live = useRef({ cpu: 0, mem: 0, load1: 0, hostPsiCpu: 0, cgroupPsiCpu: 0 });
  const prevWrite = useRef({ sectors: 0, t: 0 });

  useEffect(() => {
    const es = new EventSource(`${SYSMON_URL}/api/stream`);
    es.addEventListener("metrics", (ev) => {
      try {
        const p = JSON.parse((ev as MessageEvent).data) as Payload;
        const samples = p.snapshot?.samples ?? [];
        const hpCpu = val(samples, "host.psi.cpu.some.avg10");
        const hpIo = val(samples, "host.psi.io.some.avg10");
        const cpCpu = val(samples, "cgroup.web.psi.cpu.some.avg10");
        const cpIo = val(samples, "cgroup.web.psi.io.some.avg10");
        live.current = {
          cpu: val(samples, "cpu.usage_percent"),
          mem: val(samples, "memory.usage_percent"),
          load1: val(samples, "loadavg.1"),
          hostPsiCpu: hpCpu,
          cgroupPsiCpu: cpCpu,
        };
        setCpu(live.current.cpu);
        setMem(live.current.mem);
        setLoad1(live.current.load1);
        const now = Date.now();
        const sectors = writeSectors(samples);
        let mbs = 0;
        if (prevWrite.current.t > 0 && now > prevWrite.current.t) {
          const dt = (now - prevWrite.current.t) / 1000;
          const delta = Math.max(0, sectors - prevWrite.current.sectors);
          mbs = (delta * 512) / (1024 * 1024) / dt;
        }
        prevWrite.current = { sectors, t: now };
        setHostPsiCpu(hpCpu);
        setHostPsiIo(hpIo);
        setWriteMbs(mbs);
        setCgroupPsiCpu(cpCpu);
        setCgroupPsiIo(cpIo);
        setFiring(p.firing ?? 0);
        setPressure(cpCpu > 20);
        setHostHistCpu((h) => push(h, hpCpu));
        setHostHistIo((h) => push(h, hpIo));
        setHostHistWrite((h) => push(h, mbs));
        setCgHistCpu((h) => push(h, cpCpu));
        setCgHistIo((h) => push(h, cpIo));
        setSse("connected");
      } catch {
        /* ignore */
      }
    });
    es.onerror = () => setSse("disconnected");
    es.onopen = () => setSse("connected");
    return () => es.close();
  }, []);

  useEffect(() => {
    const t = setInterval(async () => {
      try {
        const s = await fetch("/api/stats").then((r) => r.json());
        setStats({
          p50: s.latencyMs?.p50 ?? 0,
          p99: s.latencyMs?.p99 ?? 0,
          recv: s.recv ?? 0,
          done: s.done ?? s.rps ?? 0,
          wait: s.wait ?? s.inFlight ?? 0,
          totalRecv: s.totalRecv ?? 0,
          totalDone: s.totalDone ?? 0,
          offer: s.offer ?? 0,
          backlog: s.backlog ?? 0,
          errorRate: s.errorRate ?? 0,
        });
        setP99Hist((h) => push(h, s.latencyMs?.p99 ?? 0));
      } catch {
        /* ignore */
      }
      try {
        const b = await fetch("/api/bench/status").then((r) => r.json());
        setBench(b);
      } catch {
        /* ignore */
      }
    }, 1000);
    return () => clearInterval(t);
  }, []);

  async function start() {
    await fetch("/api/bench/start", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ scenario }),
    });
  }
  async function stop() {
    await fetch("/api/bench/stop", { method: "POST" });
  }

  const benchLabel = (() => {
    if (bench.running) return `${bench.scenario || "?"} / ${bench.phase || "…"}`;
    if (stats.wait > 0) {
      const n = stats.wait;
      return bench.scenario ? `${bench.scenario} / draining (${n})` : `draining (${n})`;
    }
    return "idle";
  })();

  return (
    <div className={styles.wrap}>
      <header className={styles.bar}>
        <strong>demo-sysmon</strong>
        <span className={sse === "connected" ? styles.ok : styles.bad}>SSE: {sse}</span>
        <span>bench: {benchLabel}</span>
        <span>recv: {stats.recv}</span>
        <span>done: {stats.done}</span>
        <span>wait: {stats.wait}</span>
        {(stats.totalRecv > 0 || stats.totalDone > 0) && (
          <>
            <span>total recv: {stats.totalRecv}</span>
            <span>total done: {stats.totalDone}</span>
          </>
        )}
        <span>alerts firing: {firing}</span>
      </header>

      <div className={styles.grid}>
        <section className={styles.card}>
          <h2>Host</h2>
          <dl>
            <dt>CPU</dt><dd>{cpu.toFixed(1)}%</dd>
            <dt>Memory</dt><dd>{mem.toFixed(1)}%</dd>
            <dt>Load 1m</dt><dd>{load1.toFixed(2)}</dd>
            <dt>PSI cpu</dt><dd>{hostPsiCpu.toFixed(2)}</dd>
            <dt>PSI io</dt><dd>{hostPsiIo.toFixed(2)}</dd>
            <dt>Write</dt><dd>{writeMbs.toFixed(2)} MB/s</dd>
          </dl>
          <div className={styles.sparks}>
            <div className={styles.spark}>
              <span>PSI cpu</span>
              <Sparkline values={hostHistCpu} color="#6ee7b7" />
            </div>
            <div className={styles.spark}>
              <span>PSI io</span>
              <Sparkline values={hostHistIo} color="#047857" />
            </div>
            <div className={styles.spark}>
              <span>Write</span>
              <Sparkline values={hostHistWrite} color="#86efac" suffix=" MB/s" />
            </div>
          </div>
        </section>

        <section className={styles.card}>
          <h2>Web container</h2>
          <dl>
            <dt>cgroup PSI cpu</dt><dd>{cgroupPsiCpu.toFixed(2)}</dd>
            <dt>cgroup PSI io</dt><dd>{cgroupPsiIo.toFixed(2)}</dd>
          </dl>
          {pressure && <p className={styles.warn}>Pressure detected</p>}
          <div className={styles.sparks}>
            <div className={styles.spark}>
              <span>PSI cpu</span>
              <Sparkline values={cgHistCpu} color="#fde68a" />
            </div>
            <div className={styles.spark}>
              <span>PSI io</span>
              <Sparkline values={cgHistIo} color="#d97706" />
            </div>
          </div>
        </section>

        <section className={styles.card}>
          <h2>Next.js latency</h2>
          <dl>
            <dt>p50</dt><dd>{stats.p50.toFixed(1)} ms</dd>
            <dt>p99</dt><dd>{stats.p99.toFixed(1)} ms</dd>
            <dt>recv</dt><dd>{stats.recv}</dd>
            <dt>done</dt><dd>{stats.done}</dd>
            <dt>wait</dt><dd>{stats.wait}</dd>
            {(stats.totalRecv > 0 || stats.totalDone > 0) && (
              <>
                <dt>total recv</dt><dd>{stats.totalRecv}</dd>
                <dt>total done</dt><dd>{stats.totalDone}</dd>
              </>
            )}
            <dt>Error</dt><dd>{(stats.errorRate * 100).toFixed(1)}%</dd>
            <dt>offer</dt><dd className={stats.offer > CPU_CAP ? styles.hot : undefined}>{stats.offer.toFixed(1)} / {CPU_CAP.toFixed(1)}</dd>
            <dt>backlog</dt><dd>{stats.backlog.toFixed(1)} CPU-s</dd>
          </dl>
          <div className={styles.work}>
            <span>Work</span>
            <div className={styles.workTrack}>
              <div
                className={`${styles.workFill} ${stats.offer > CPU_CAP ? styles.workOver : ""}`}
                style={{ width: `${Math.min(100, (stats.offer / CPU_CAP) * 100)}%` }}
              />
            </div>
          </div>
          <div className={styles.sparks}>
            <div className={styles.spark}>
              <span>p99</span>
              <Sparkline values={p99Hist} color="#93c5fd" suffix=" ms" />
            </div>
          </div>
        </section>
      </div>

      <section className={styles.controls}>
        <div className={styles.scenario}>
          <select value={scenario} onChange={(e) => setScenario(e.target.value)}>
            <option value="ramp">ramp</option>
            <option value="cpu-burst">cpu-burst</option>
            <option value="io-burst">io-burst</option>
          </select>
          <p>{HINT[scenario]}</p>
        </div>
        <button type="button" onClick={start}>負荷開始</button>
        <button type="button" onClick={stop}>停止</button>
      </section>
    </div>
  );
}

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

function push(buf: number[], v: number): number[] {
  const next = buf.length >= CAP ? buf.slice(buf.length - CAP + 1) : buf.slice();
  next.push(v);
  return next;
}

function val(samples: Sample[], name: string): number {
  return samples.find((s) => s.name === name)?.value ?? 0;
}

export function Dashboard() {
  const [sse, setSse] = useState<"connected" | "disconnected">("disconnected");
  const [cpu, setCpu] = useState(0);
  const [mem, setMem] = useState(0);
  const [load1, setLoad1] = useState(0);
  const [hostPsiCpu, setHostPsiCpu] = useState(0);
  const [hostPsiIo, setHostPsiIo] = useState(0);
  const [cgroupPsiCpu, setCgroupPsiCpu] = useState(0);
  const [cgroupPsiIo, setCgroupPsiIo] = useState(0);
  const [firing, setFiring] = useState(0);
  const [pressure, setPressure] = useState(false);
  const [hostHist, setHostHist] = useState<number[]>([]);
  const [cgHist, setCgHist] = useState<number[]>([]);
  const [p99Hist, setP99Hist] = useState<number[]>([]);
  const [stats, setStats] = useState({ p50: 0, p99: 0, rps: 0, errorRate: 0, inFlight: 0 });
  const [bench, setBench] = useState<Bench>({});
  const [scenario, setScenario] = useState("ramp");
  const live = useRef({ cpu: 0, mem: 0, load1: 0, hostPsiCpu: 0, cgroupPsiCpu: 0 });

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
        setHostPsiCpu(hpCpu);
        setHostPsiIo(hpIo);
        setCgroupPsiCpu(cpCpu);
        setCgroupPsiIo(cpIo);
        setFiring(p.firing ?? 0);
        setPressure(cpCpu > 20);
        setHostHist((h) => push(h, hpCpu));
        setCgHist((h) => push(h, cpIo));
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
          rps: s.rps ?? 0,
          errorRate: s.errorRate ?? 0,
          inFlight: s.inFlight ?? 0,
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
    if (stats.inFlight > 0) {
      const n = stats.inFlight;
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
          </dl>
          <Sparkline values={hostHist} />
        </section>

        <section className={styles.card}>
          <h2>Web container</h2>
          <dl>
            <dt>cgroup PSI cpu</dt><dd>{cgroupPsiCpu.toFixed(2)}</dd>
            <dt>cgroup PSI io</dt><dd>{cgroupPsiIo.toFixed(2)}</dd>
          </dl>
          {pressure && <p className={styles.warn}>Pressure detected</p>}
          <Sparkline values={cgHist} color="#fbbf24" />
        </section>

        <section className={styles.card}>
          <h2>Next.js latency</h2>
          <dl>
            <dt>p50</dt><dd>{stats.p50.toFixed(1)} ms</dd>
            <dt>p99</dt><dd>{stats.p99.toFixed(1)} ms</dd>
            <dt>RPS</dt><dd>{stats.rps.toFixed(1)}</dd>
            <dt>Error</dt><dd>{(stats.errorRate * 100).toFixed(1)}%</dd>
          </dl>
          <Sparkline values={p99Hist} color="#93c5fd" />
        </section>
      </div>

      <section className={styles.controls}>
        <select value={scenario} onChange={(e) => setScenario(e.target.value)}>
          <option value="ramp">ramp</option>
          <option value="cpu-burst">cpu-burst</option>
          <option value="io-burst">io-burst</option>
        </select>
        <button type="button" onClick={start}>負荷開始</button>
        <button type="button" onClick={stop}>停止</button>
      </section>
    </div>
  );
}

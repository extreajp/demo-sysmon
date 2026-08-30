import { Worker } from "worker_threads";
import path from "path";
import { denyIfDisabled, timed } from "@/lib/guard";

export const dynamic = "force-dynamic";

export async function GET(req: Request) {
  const denied = denyIfDisabled();
  if (denied) return denied;
  const ms = Math.min(10000, Math.max(1, Number(new URL(req.url).searchParams.get("ms") || "200")));
  const n = Math.max(1, Number(process.env.DEMO_CPU_WORKERS || "2"));
  const file = process.env.CPU_WORKER_PATH || path.join(process.cwd(), "cpu-worker.js");
  return timed(async () => {
    await Promise.all(
      Array.from({ length: n }, () =>
        new Promise<number>((resolve, reject) => {
          const w = new Worker(file, { workerData: { ms } });
          w.once("message", (v) => resolve(Number(v) || 0));
          w.once("error", reject);
        }),
      ),
    );
    return { workers: n, ms };
  }, ms * n);
}

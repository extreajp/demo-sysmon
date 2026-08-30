import fs from "fs";
import path from "path";

export function loadEnabled(): boolean {
  return process.env.DEMO_LOAD_ENABLED === "1";
}

export function busyWork(ms: number) {
  const end = Date.now() + ms;
  let x = 0;
  while (Date.now() < end) {
    x += Math.sqrt((x + 1) % 1000);
  }
  return x;
}

export function holdMemory(mb: number, holdSec: number): Promise<number> {
  const buf = Buffer.alloc(Math.max(1, mb) * 1024 * 1024, 1);
  return new Promise((resolve) => {
    setTimeout(() => {
      buf.fill(0);
      resolve(buf.length);
    }, holdSec * 1000);
  });
}

export function syncIO(mb: number): number {
  const n = Math.max(1, mb) * 1024 * 1024;
  const p = path.join("/tmp", "sysmon-io-" + process.pid);
  const fd = fs.openSync(p, "w");
  try {
    const chunk = Buffer.alloc(64 * 1024, 7);
    let left = n;
    while (left > 0) {
      const w = Math.min(left, chunk.length);
      fs.writeSync(fd, chunk, 0, w);
      left -= w;
    }
    fs.fsyncSync(fd);
  } finally {
    fs.closeSync(fd);
    try {
      fs.unlinkSync(p);
    } catch {
      /* ignore */
    }
  }
  return n;
}

import { NextResponse } from "next/server";
import { begin, end } from "./stats";
import { loadEnabled } from "./load";

export function denyIfDisabled() {
  if (!loadEnabled()) {
    return NextResponse.json({ error: "load disabled" }, { status: 403 });
  }
  return null;
}

export async function timed(fn: () => Promise<unknown> | unknown, cpuMs = 0) {
  const t0 = begin(cpuMs);
  try {
    const v = await fn();
    end(t0, false);
    return NextResponse.json({ ok: true, result: v });
  } catch (e) {
    end(t0, true);
    const msg = e instanceof Error ? e.message : String(e);
    return NextResponse.json({ ok: false, error: msg }, { status: 500 });
  }
}

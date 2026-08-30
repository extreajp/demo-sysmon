import { NextResponse } from "next/server";
import { loadgen } from "@/lib/loadgen";

export const dynamic = "force-dynamic";

export async function POST() {
  const r = await loadgen("/stop", { method: "POST" });
  const text = await r.text();
  return new NextResponse(text, { status: r.status, headers: { "content-type": "application/json" } });
}

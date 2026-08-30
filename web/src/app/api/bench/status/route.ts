import { NextResponse } from "next/server";
import { loadgen } from "@/lib/loadgen";

export const dynamic = "force-dynamic";

export async function GET() {
  const r = await loadgen("/status");
  const text = await r.text();
  return new NextResponse(text, { status: r.status, headers: { "content-type": "application/json" } });
}

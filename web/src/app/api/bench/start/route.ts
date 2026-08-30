import { NextResponse } from "next/server";
import { loadgen } from "@/lib/loadgen";

export const dynamic = "force-dynamic";

export async function POST(req: Request) {
  const body = await req.json().catch(() => ({}));
  const r = await loadgen("/run", { method: "POST", body: JSON.stringify(body) });
  const text = await r.text();
  return new NextResponse(text, { status: r.status, headers: { "content-type": "application/json" } });
}

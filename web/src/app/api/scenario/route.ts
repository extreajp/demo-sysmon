import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

let phase = "";

export function GET() {
  return NextResponse.json({ phase });
}

export async function POST(req: Request) {
  const body = await req.json().catch(() => ({}));
  phase = typeof body.phase === "string" ? body.phase : String(body.phase || "");
  return NextResponse.json({ ok: true, phase });
}

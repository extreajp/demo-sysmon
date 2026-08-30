import { NextResponse } from "next/server";
import { snapshot } from "@/lib/stats";

export const dynamic = "force-dynamic";

export function GET() {
  return NextResponse.json(snapshot());
}

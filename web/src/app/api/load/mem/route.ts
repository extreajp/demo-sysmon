import { holdMemory } from "@/lib/load";
import { denyIfDisabled, timed } from "@/lib/guard";

export const dynamic = "force-dynamic";

export async function GET(req: Request) {
  const denied = denyIfDisabled();
  if (denied) return denied;
  const q = new URL(req.url).searchParams;
  const mb = Math.min(256, Math.max(1, Number(q.get("mb") || "128")));
  const hold = Math.min(30, Math.max(0, Number(q.get("hold") || "5")));
  return timed(() => holdMemory(mb, hold));
}

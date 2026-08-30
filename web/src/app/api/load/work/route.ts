import { busyWork } from "@/lib/load";
import { denyIfDisabled, timed } from "@/lib/guard";

export const dynamic = "force-dynamic";

export async function GET(req: Request) {
  const denied = denyIfDisabled();
  if (denied) return denied;
  const ms = Math.min(5000, Math.max(1, Number(new URL(req.url).searchParams.get("ms") || "50")));
  return timed(() => busyWork(ms));
}

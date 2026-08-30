const BASE = process.env.LOADGEN_URL || "http://loadgen:8000";

export async function loadgen(path: string, init?: RequestInit): Promise<Response> {
  return fetch(BASE + path, {
    ...init,
    headers: { "content-type": "application/json", ...(init?.headers || {}) },
  });
}

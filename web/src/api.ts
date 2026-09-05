export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function api<T = any>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch("/api/v1" + path, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(opts.headers || {}) },
    ...opts,
  });
  if (res.status === 204) return null as T;
  const ct = res.headers.get("content-type") || "";
  const body = ct.includes("json") ? await res.json().catch(() => null) : await res.text();
  if (!res.ok) {
    const msg = (body && (body.detail || body.title)) || res.statusText;
    throw new ApiError(res.status, msg);
  }
  return body as T;
}

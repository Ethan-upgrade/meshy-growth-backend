const ADMIN_API_URL = process.env.ADMIN_API_URL ?? "http://admin-api:8080";

export async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${ADMIN_API_URL}${path}`, {
    ...init,
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`admin-api ${path} failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

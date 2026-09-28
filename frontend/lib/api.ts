// Thin fetch wrapper around the Go backend. Every service module (once they
// exist, e.g. lib/services/students.ts) should go through this rather than
// calling fetch() directly, so the base URL and error handling live in one
// place.

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method: "GET",
    credentials: "include",
  });

  if (!res.ok) {
    throw new Error(`API GET ${path} failed with status ${res.status}`);
  }

  return res.json() as Promise<T>;
}

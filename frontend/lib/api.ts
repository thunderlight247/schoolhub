// Thin fetch wrapper around the Go backend. Every service module should go
// through this rather than calling fetch() directly, so the base URL,
// credentials handling, and error shaping live in one place.
//
// All requests send credentials: the backend uses cookie sessions (M4), so
// the browser must attach the session cookie on every call and on the CORS
// preflight the backend echoes FRONTEND_URL back (see the backend config
// comments on why "*" is not an option with credentialed requests).

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

/** Structured error from the backend's {"error": {code, message, details}} shape. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }

  get isUnauthenticated(): boolean {
    return this.status === 401;
  }

  get isForbidden(): boolean {
    return this.status === 403;
  }
}

interface ApiErrorBody {
  error?: { code?: string; message?: string };
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method,
    credentials: "include",
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let code = `HTTP_${res.status}`;
    let message = `API ${method} ${path} failed with status ${res.status}`;
    try {
      const parsed = (await res.json()) as ApiErrorBody;
      if (parsed.error?.code) code = parsed.error.code;
      if (parsed.error?.message) message = parsed.error.message;
    } catch {
      // non-JSON error body — keep the generic message
    }
    throw new ApiError(res.status, code, message);
  }

  // 204 No Content (e.g. logout) and other empty bodies must not hit res.json().
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

export function apiGet<T>(path: string): Promise<T> {
  return request<T>("GET", path);
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>("POST", path, body);
}

export function apiPatch<T>(path: string, body?: unknown): Promise<T> {
  return request<T>("PATCH", path, body);
}

export function apiPut<T>(path: string, body?: unknown): Promise<T> {
  return request<T>("PUT", path, body);
}

export function apiDelete<T>(path: string): Promise<T> {
  return request<T>("DELETE", path);
}

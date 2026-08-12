const BASE_URL = import.meta.env.VITE_BACKEND_URL;

if (!BASE_URL) {
  throw new Error("VITE_BACKEND_URL is not set in frontend/.env");
}

function getTokenKey(): string {
  const adminRole = localStorage.getItem("admin_role");
  if (adminRole === "admin") return "admin_token";
  const coachRole = localStorage.getItem("coach_role");
  if (coachRole === "coach") return "coach_token";
  return "admin_token";
}

/**
 * Shared fetch wrapper that attaches a JWT and handles errors.
 * @param tokenKey - localStorage key for the token (auto-detected if not provided)
 */
export async function apiFetch<T = unknown>(
  url: string,
  options: RequestInit = {},
  tokenKey?: string
): Promise<T> {
  const key = tokenKey || getTokenKey();
  const token = localStorage.getItem(key);

  const headers = new Headers(options.headers);

  if (!(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${url}`, { ...options, headers });

  const payload = await res.json().catch(() => ({ error: "Invalid response" }));

  if (!res.ok) {
    const message =
      typeof payload === "object" && payload !== null && "error" in payload
        ? (payload as { error: string }).error
        : `Request failed with status ${res.status}`;
    const err = new Error(message);
    (err as Error & { payload?: unknown }).payload = payload;
    throw err;
  }

  return payload as T;
}

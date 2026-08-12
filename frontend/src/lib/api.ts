const BASE_URL = import.meta.env.VITE_BACKEND_URL;

if (!BASE_URL) {
  throw new Error("VITE_BACKEND_URL is not set in frontend/.env");
}

/**
 * Shared fetch wrapper that attaches a JWT and handles errors.
 * @param tokenKey - localStorage key for the token (default: "admin_token")
 */
export async function apiFetch<T = unknown>(
  url: string,
  options: RequestInit = {},
  tokenKey: string = "admin_token"
): Promise<T> {
  const token = localStorage.getItem(tokenKey);

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

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

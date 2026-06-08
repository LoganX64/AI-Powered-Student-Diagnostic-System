const BASE_URL = import.meta.env.VITE_BACKEND_URL;

/**
 * Shared fetch wrapper that attaches the admin JWT and handles errors.
 */
export async function apiFetch<T = unknown>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  const token = localStorage.getItem("admin_token");

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
    throw new Error(message);
  }

  return payload as T;
}

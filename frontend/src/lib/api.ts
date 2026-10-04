import { TOKEN_KEYS, type Role } from "@/lib/token";

const BASE_URL = import.meta.env.VITE_BACKEND_URL;

if (!BASE_URL) {
  // No throw at module eval: a throw here would prevent main.tsx from running and
  // leave the #splash-loader spinner forever. Surface the cause on the splash
  // instead so the page is blank-free, and fail every request with a clear error.
  const splash = document.getElementById("splash-loader");
  if (splash) {
    splash.innerHTML =
      '<div style="max-width:34rem;text-align:center;padding:0 1rem">' +
      '<p style="color:#b91c1c;font-weight:600;margin:0 0 8px">Configuration error</p>' +
      '<p style="color:#6b7280;font-size:14px;margin:0">VITE_BACKEND_URL is not set in frontend/.env. ' +
      'Copy frontend/.env.example to frontend/.env and set the backend URL, then restart.</p></div>';
    splash.classList.remove("hidden");
  }
  (window as unknown as { __API_CONFIG_ERROR__?: boolean }).__API_CONFIG_ERROR__ = true;
}

/**
 * Error carrying the HTTP status, so callers can branch on 401/403/404 instead of
 * matching on the message text.
 */
export class ApiError extends Error {
  status: number;
  payload?: unknown;
  constructor(message: string, status: number, payload?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

/**
 * Resolves the role from an API path. The path already encodes the role, so this
 * stays correct when several roles are signed in on one machine — reading a
 * single named role's key, rather than scanning storage for "who is active".
 *
 * Returns null for shared endpoints (/auth/*), where the caller must supply the
 * role via `tokenKey`.
 */
function roleFromUrl(url: string): Role | null {
  if (url.startsWith("/super-admin")) return "super_admin";
  if (url.startsWith("/coach")) return "coach";
  if (url.startsWith("/student")) return "student";
  if (url.startsWith("/admin")) return "admin";
  return null;
}

function resolveTokenKey(url: string): string {
  const role = roleFromUrl(url);
  return role ? TOKEN_KEYS[role] : TOKEN_KEYS.admin;
}

function handleUnauthorized(tokenKey: string) {
  const role = (Object.keys(TOKEN_KEYS) as Role[]).find(
    (r) => TOKEN_KEYS[r] === tokenKey,
  ) ?? "admin";
  // Only the session that actually failed is cleared, so other roles signed in on
  // the same machine stay valid.
  localStorage.removeItem(tokenKey);
  if (role === "student") localStorage.removeItem("student_code");
  window.dispatchEvent(new Event("role-change"));
  window.location.replace(
    role === "admin"
      ? "/admin-signin"
      : role === "coach"
        ? "/coach-signin"
        : role === "student"
          ? "/student-login"
          : "/super-admin-signin",
  );
}

/**
 * Shared fetch wrapper that attaches a JWT and handles errors.
 * @param tokenKey - localStorage key for the token. Defaults to the role implied
 *   by `url`; pass it explicitly for shared endpoints like /auth/profile.
 */
export async function apiFetch<T = unknown>(
  url: string,
  options: RequestInit = {},
  tokenKey?: string,
): Promise<T> {
  if (!BASE_URL) {
    throw new ApiError(
      "VITE_BACKEND_URL is not set in frontend/.env",
      0,
    );
  }
  const key = tokenKey || resolveTokenKey(url);
  const token = localStorage.getItem(key);

  const headers = new Headers(options.headers);

  if (!(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(`${BASE_URL}${url}`, { ...options, headers });

  const payload = await res.json().catch(() => ({ error: "Invalid response" }));

  if (!res.ok) {
    const message =
      typeof payload === "object" && payload !== null && "error" in payload
        ? (payload as { error: string }).error
        : `Request failed with status ${res.status}`;
    const err = new ApiError(message, res.status, payload);
    if (res.status === 401) handleUnauthorized(key);
    throw err;
  }

  return payload as T;
}
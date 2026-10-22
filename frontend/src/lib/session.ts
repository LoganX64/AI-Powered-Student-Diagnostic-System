export type Role = "admin" | "coach" | "student" | "super_admin";

export type Session = {
  role: Role;
  roles: Role[];
};

const BASE_URL = import.meta.env.VITE_BACKEND_URL as string | undefined;
const CACHE_TTL_MS = 30_000;

const VALID_ROLES: Role[] = ["admin", "coach", "super_admin", "student"];

let cached: { value: Session | null; at: number } | null = null;
let inFlight: Promise<Session | null> | null = null;

function isRole(value: unknown): value is Role {
  return typeof value === "string" && VALID_ROLES.includes(value as Role);
}

async function requestSession(): Promise<Session | null> {
  if (!BASE_URL) {
    throw new Error("VITE_BACKEND_URL is not set in frontend/.env");
  }

  const res = await fetch(`${BASE_URL}/auth/session`, {
    credentials: "include",
    cache: "no-store",
    headers: { Accept: "application/json" },
  });

  if (res.status === 401) {
    return null;
  }
  if (!res.ok) {
    throw new Error(`Session check failed with status ${res.status}`);
  }

  const payload: unknown = await res.json().catch(() => null);
  if (
    typeof payload !== "object" ||
    payload === null ||
    !("authenticated" in payload) ||
    payload.authenticated !== true
  ) {
    return null;
  }

  const role = "role" in payload && isRole(payload.role) ? payload.role : null;
  if (!role) return null;

  const rawRoles =
    "roles" in payload && Array.isArray(payload.roles)
      ? payload.roles.filter(isRole)
      : [];
  const roles = Array.from(new Set<Role>([role, ...rawRoles]));

  return { role, roles };
}

/**
 * Resolves the current browser session from the single /auth/session endpoint.
 * Concurrent callers share one in-flight request, and a short cache keeps
 * StrictMode remounts and normal navigation from refetching.
 *
 * Use force=true after login/logout so a stale cached session cannot win.
 */
export async function fetchSession(force = false): Promise<Session | null> {
  if (!force && cached && Date.now() - cached.at < CACHE_TTL_MS) {
    return cached.value;
  }

  if (inFlight) {
    if (!force) return inFlight;
    try {
      await inFlight;
    } catch {
      // The forced request below gets a fresh chance.
    }
  }

  const request = requestSession();
  inFlight = request;

  try {
    const value = await request;
    cached = { value, at: Date.now() };
    return value;
  } finally {
    if (inFlight === request) {
      inFlight = null;
    }
  }
}

/** Drops the cached result; the next fetch hits the backend again. */
export function invalidateSession(): void {
  cached = null;
}

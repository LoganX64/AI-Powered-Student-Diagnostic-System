export type Role = "admin" | "coach" | "student" | "super_admin";

export type Session = { role: Role };

const BASE_URL = import.meta.env.VITE_BACKEND_URL as string;

const PROBE_URLS: Record<Role, string> = {
  admin: "/admin/notifications/unread-count",
  coach: "/coach/notifications/unread-count",
  super_admin: "/super-admin/stats",
  student: "/student/assignments",
};

/**
 * True when the browser holds a valid HttpOnly session cookie for `role`.
 * Uses a cheap GET against a role-gated endpoint; a cookie for a different
 * role fails the role check server-side rather than being mistaken for one.
 */
export async function probeRole(role: Role): Promise<boolean> {
  try {
    const res = await fetch(`${BASE_URL}${PROBE_URLS[role]}`, {
      credentials: "include",
      headers: { "X-Role": role },
    });
    return res.ok;
  } catch {
    return false;
  }
}

/**
 * First role with a valid session cookie, or null. Replaces the old
 * mostRecentSession() localStorage scan now that tokens are HttpOnly.
 */
export async function probeSession(): Promise<Session | null> {
  for (const role of ["admin", "coach", "super_admin", "student"] as Role[]) {
    if (await probeRole(role)) return { role };
  }
  return null;
}

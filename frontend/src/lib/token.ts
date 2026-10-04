export type Role = "admin" | "coach" | "student" | "super_admin";

interface TokenPayload {
  user_id: number;
  role: Role;
  student_id: number;
  exp: number;
  iat: number;
}

export const TOKEN_KEYS: Record<Role, string> = {
  admin: "admin_token",
  coach: "coach_token",
  student: "student_token",
  super_admin: "super_admin_token",
};

export function getTokenPayload(token: string): TokenPayload | null {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    if (typeof payload.role !== "string") return null;
    return payload as TokenPayload;
  } catch {
    return null;
  }
}

export function isTokenExpired(token: string): boolean {
  const payload = getTokenPayload(token);
  if (!payload) return true;
  return payload.exp * 1000 < Date.now();
}

export type Session = { role: Role; iat: number };

/**
 * The most recently authenticated valid session, or null when signed out.
 *
 * Ranks by each token's own `iat` rather than by TOKEN_KEYS declaration order.
 * Order-based scanning is the bug Phase 3.2 fixed: with two roles signed in it
 * handed a super-admin the admin JWT. The highest `iat` is the session the user
 * created last, which is the one they mean when they open "/".
 */
export function mostRecentSession(): Session | null {
  let best: Session | null = null;

  for (const role of Object.keys(TOKEN_KEYS) as Role[]) {
    const token = localStorage.getItem(TOKEN_KEYS[role]);
    if (!token || isTokenExpired(token)) continue;

    const payload = getTokenPayload(token);
    // A token stored under the wrong key is not a session for that role.
    if (!payload || payload.role !== role) continue;
    if (typeof payload.iat !== "number") continue;

    if (!best || payload.iat > best.iat) best = { role, iat: payload.iat };
  }

  return best;
}

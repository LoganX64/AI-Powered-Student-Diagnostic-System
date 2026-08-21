export type Role = "admin" | "coach" | "student";

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

/**
 * Derives the active user role from the stored JWTs (not from stale
 * localStorage flag mirrors). Returns null when no valid token is present.
 */
export function getActiveRole(): Role | null {
  for (const role of Object.keys(TOKEN_KEYS) as Role[]) {
    const token = localStorage.getItem(TOKEN_KEYS[role]);
    if (token) {
      const payload = getTokenPayload(token);
      if (payload && payload.role === role) return role;
    }
  }
  return null;
}

export function isTokenExpired(token: string): boolean {
  const payload = getTokenPayload(token);
  if (!payload) return true;
  return payload.exp * 1000 < Date.now();
}

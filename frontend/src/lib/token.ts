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

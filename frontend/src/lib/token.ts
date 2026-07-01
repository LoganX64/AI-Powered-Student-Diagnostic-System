export interface TokenPayload {
  user_id: number;
  role: "admin" | "coach" | "student";
  student_id: number;
  exp: number;
  iat: number;
}

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

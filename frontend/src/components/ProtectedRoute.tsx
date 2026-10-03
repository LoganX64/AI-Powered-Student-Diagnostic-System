import { Navigate, Outlet, useLocation } from "react-router-dom";
import { TOKEN_KEYS, getTokenPayload, isTokenExpired, type Role } from "@/lib/token";
import { roleFromPath, ROLE_REDIRECT_MAP } from "@/config/routes";

function getRoleFromToken(tokenKey: string): Role | null {
  const token = localStorage.getItem(tokenKey);
  if (!token) return null;
  const payload = getTokenPayload(token);
  if (!payload) return null;
  if (payload.role === "admin" || payload.role === "coach" || payload.role === "student" || payload.role === "super_admin") {
    return payload.role;
  }
  return null;
}

// Only the expired session is cleared, so other roles signed in on the same
// machine stay signed in.
function clearExpiredToken(tokenKey: string) {
  localStorage.removeItem(tokenKey);
  if (tokenKey === TOKEN_KEYS.student) localStorage.removeItem("student_code");
}

function isAuthenticated(role: Role): boolean {
  switch (role) {
    case "admin": {
      const tokenKey = "admin_token";
      const token = localStorage.getItem(tokenKey);
      if (!token) return false;
      const tokenRole = getRoleFromToken(tokenKey);
      if (tokenRole !== "admin") return false;
      if (isTokenExpired(token)) {
        clearExpiredToken(tokenKey);
        return false;
      }
      return true;
    }
    case "coach": {
      const tokenKey = "coach_token";
      const token = localStorage.getItem(tokenKey);
      if (!token) return false;
      const tokenRole = getRoleFromToken(tokenKey);
      if (tokenRole !== "coach") return false;
      if (isTokenExpired(token)) {
        clearExpiredToken(tokenKey);
        return false;
      }
      return true;
    }
    case "super_admin": {
      const tokenKey = "super_admin_token";
      const token = localStorage.getItem(tokenKey);
      if (!token) return false;
      const tokenRole = getRoleFromToken(tokenKey);
      if (tokenRole !== "super_admin") return false;
      if (isTokenExpired(token)) {
        clearExpiredToken(tokenKey);
        return false;
      }
      return true;
    }
    case "student": {
      const tokenKey = "student_token";
      const token = localStorage.getItem(tokenKey);
      if (!token) return false;
      const tokenRole = getRoleFromToken(tokenKey);
      if (tokenRole !== "student") return false;
      if (isTokenExpired(token)) {
        clearExpiredToken(tokenKey);
        return false;
      }
      return true;
    }
    default:
      return false;
  }
}

export function ProtectedRoute() {
  const { pathname } = useLocation();
  const requiredRole = roleFromPath(pathname);

  if (!requiredRole) return <Navigate to="/" replace />;
  if (isAuthenticated(requiredRole)) return <Outlet />;
  return <Navigate to={ROLE_REDIRECT_MAP[requiredRole]} replace />;
}

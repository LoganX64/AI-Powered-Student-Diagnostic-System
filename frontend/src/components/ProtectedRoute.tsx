import { Navigate, Outlet, useLocation } from "react-router-dom";
import { getTokenPayload, isTokenExpired } from "@/lib/token";
import { ROLE_CHANGE_EVENT } from "@/hooks/useRole";

type Role = "admin" | "coach" | "student";

const REDIRECT_MAP: Record<Role, string> = {
  admin: "/admin-signin",
  coach: "/coach-signin",
  student: "/student-login",
};

function detectRoleFromPath(pathname: string): Role | null {
  if (pathname.startsWith("/admin")) return "admin";
  if (pathname.startsWith("/coach")) return "coach";
  if (["/dashboard", "/instructions", "/quiz", "/submitted"].includes(pathname)) return "student";
  return null;
}

function getRoleFromToken(tokenKey: string): Role | null {
  const token = localStorage.getItem(tokenKey);
  if (!token) return null;
  const payload = getTokenPayload(token);
  if (!payload) return null;
  if (payload.role === "admin" || payload.role === "coach" || payload.role === "student") {
    return payload.role;
  }
  return null;
}

function clearExpiredToken(tokenKey: string) {
  if (tokenKey === "student_token") {
    localStorage.removeItem("student_token");
    localStorage.removeItem("student_code");
  } else {
    localStorage.removeItem("admin_token");
    localStorage.removeItem("admin_role");
  }
  window.dispatchEvent(new Event(ROLE_CHANGE_EVENT));
}

function isAuthenticated(role: Role): boolean {
  switch (role) {
    case "admin":
    case "coach": {
      const tokenKey = "admin_token";
      const token = localStorage.getItem(tokenKey);
      if (!token) return false;
      const tokenRole = getRoleFromToken(tokenKey);
      if (tokenRole !== role) return false;
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
  const requiredRole = detectRoleFromPath(pathname);

  if (!requiredRole) return <Navigate to="/" replace />;
  if (isAuthenticated(requiredRole)) return <Outlet />;
  return <Navigate to={REDIRECT_MAP[requiredRole]} replace />;
}

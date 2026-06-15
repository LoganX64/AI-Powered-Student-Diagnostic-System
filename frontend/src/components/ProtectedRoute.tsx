import { Navigate, Outlet, useLocation } from "react-router-dom";
import { isTokenExpired } from "@/lib/token";

type Role = "admin" | "coach" | "student";

const REDIRECT_MAP: Record<Role, string> = {
  admin: "/admin-signin",
  coach: "/coach-signin",
  student: "/student-login",
};

function detectRole(pathname: string): Role | null {
  if (pathname.startsWith("/admin")) return "admin";
  if (pathname.startsWith("/coach")) return "coach";
  // Student-protected routes
  if (["/instructions", "/quiz", "/submitted"].includes(pathname)) return "student";
  return null;
}

function clearExpiredToken(role: Role) {
  if (role === "student") {
    localStorage.removeItem("student_token");
    localStorage.removeItem("student_code");
  } else {
    localStorage.removeItem("admin_token");
    localStorage.removeItem("admin_role");
  }
}

function isAuthenticated(role: Role): boolean {
  switch (role) {
    case "admin": {
      const token = localStorage.getItem("admin_token");
      const roleValue = localStorage.getItem("admin_role");
      if (!token || roleValue !== "admin") return false;
      if (isTokenExpired(token)) {
        clearExpiredToken(role);
        return false;
      }
      return true;
    }
    case "coach": {
      const token = localStorage.getItem("admin_token");
      const roleValue = localStorage.getItem("admin_role");
      if (!token || roleValue !== "coach") return false;
      if (isTokenExpired(token)) {
        clearExpiredToken(role);
        return false;
      }
      return true;
    }
    case "student": {
      const token = localStorage.getItem("student_token");
      if (!token) return false;
      if (isTokenExpired(token)) {
        clearExpiredToken(role);
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
  const role = detectRole(pathname);

  if (!role || isAuthenticated(role)) {
    return <Outlet />;
  }

  return <Navigate to={REDIRECT_MAP[role]} replace />;
}

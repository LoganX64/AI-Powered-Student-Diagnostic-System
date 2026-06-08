import { Navigate, Outlet, useLocation } from "react-router-dom";

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

function isAuthenticated(role: Role): boolean {
  switch (role) {
    case "admin": {
      const token = localStorage.getItem("admin_token");
      const roleValue = localStorage.getItem("admin_role");
      return !!token && roleValue === "admin";
    }
    case "coach": {
      const token = localStorage.getItem("admin_token");
      const roleValue = localStorage.getItem("admin_role");
      return !!token && roleValue === "coach";
    }
    case "student": {
      return !!localStorage.getItem("student_token");
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

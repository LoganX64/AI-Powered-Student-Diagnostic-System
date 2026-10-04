import type { Role } from "@/lib/token";

export const STUDENT_ROUTES = ["/dashboard", "/instructions", "/quiz", "/submitted"] as const;

export const ROLE_REDIRECT_MAP = {
  admin: "/admin-signin",
  coach: "/coach-signin",
  student: "/student-login",
  super_admin: "/super-admin-signin",
} as const;

/**
 * Resolves the active role from a pathname. Single source of truth for both
 * ProtectedRoute (access control) and useRole (UI + API prefix), so the two
 * cannot disagree when several roles are signed in on one machine.
 */
export function roleFromPath(pathname: string): Role | null {
  if (pathname.startsWith("/super-admin")) return "super_admin";
  if (pathname.startsWith("/admin")) return "admin";
  if (pathname.startsWith("/coach")) return "coach";
  if (STUDENT_ROUTES.includes(pathname as (typeof STUDENT_ROUTES)[number])) return "student";
  return null;
}

/** Returns the API path prefix for a role. */
export function prefixForRole(role: Role | null): string {
  if (role === "coach") return "/coach";
  return "/admin";
}

/** Landing page for a signed-in role. */
export function dashboardForRole(role: Role): string {
  if (role === "coach") return "/coach/dashboard";
  if (role === "student") return "/dashboard";
  if (role === "super_admin") return "/super-admin/dashboard";
  return "/admin/dashboard";
}

/**
 * Whether a sidebar nav item should render as active for the current pathname.
 *
 * Prefix match on a path-segment boundary, so a detail route still lights up its
 * parent item — /admin/students/42 highlights "/admin/students". Exact equality
 * would leave the sidebar with no active item at all on every detail page.
 * The boundary check keeps "/admin/tests" from matching "/admin/tests-archive".
 */
export function isActiveRoute(pathname: string, target: string): boolean {
  if (pathname === target) return true;
  if (!pathname.startsWith(target)) return false;
  const next = pathname.charAt(target.length);
  return next === "" || next === "/";
}

/**
 * Prefix for the role currently browsing, read straight from window.location.
 *
 * For use inside services and hooks, which have no router context. Matches what
 * useRole() returns during a client render, so a service builds /coach URLs for a
 * coach and /admin URLs for an admin even when both are signed in.
 *
 * Falls back to /admin when the path is outside the dashboard (e.g. /), which is
 * the same default prefixForRole applies.
 */
export function currentPrefix(): string {
  if (typeof window === "undefined") return "/admin";
  return prefixForRole(roleFromPath(window.location.pathname));
}

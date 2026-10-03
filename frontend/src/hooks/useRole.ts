import { useLocation } from "react-router-dom";
import { roleFromPath } from "@/config/routes";
import type { Role } from "@/lib/token";

export type { Role };

export const ROLE_CHANGE_EVENT = "role-change";

/**
 * Returns the role for the current route.
 *
 * Derived from the pathname rather than by scanning localStorage for a token:
 * when several roles are signed in on one machine, the route is what says who is
 * browsing right now. Scanning storage instead returns whichever token appears
 * first in key-declaration order, which is always `admin` once an admin session
 * exists.
 */
export function useRole(): Role | null {
  const { pathname } = useLocation();
  return roleFromPath(pathname);
}
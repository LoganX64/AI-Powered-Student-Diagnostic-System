import { useMemo } from "react";
import { useLocation } from "react-router-dom";
import type { Role } from "@/contexts/RoleContext";

export function useRole(): Role {
  const { pathname } = useLocation();
  return useMemo(() => {
    const storedRole = localStorage.getItem("admin_role") as Role | null;
    if (pathname.startsWith("/admin")) {
      return storedRole === "admin" ? "admin" : "coach";
    }
    return storedRole === "coach" ? "coach" : "admin";
  }, [pathname]);
}

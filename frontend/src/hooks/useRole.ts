import { useMemo } from "react";
import { useLocation } from "react-router-dom";
import type { Role } from "@/contexts/RoleContext";

export function useRole(): Role {
  const { pathname } = useLocation();
  return useMemo(() => {
    if (pathname.startsWith("/admin")) return "admin";
    return "coach";
  }, [pathname]);
}

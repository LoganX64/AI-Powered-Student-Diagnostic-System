import { useEffect, useState } from "react";
import { getActiveRole, type Role } from "@/lib/token";

export type { Role };

export const ROLE_CHANGE_EVENT = "role-change";

function readTokenRole(): Role | null {
  return getActiveRole();
}

export function useRole(): Role | null {
  const [role, setRole] = useState<Role | null>(() => readTokenRole());

  useEffect(() => {
    const sync = () => setRole(readTokenRole());
    window.addEventListener("storage", sync);
    window.addEventListener(ROLE_CHANGE_EVENT, sync);
    return () => {
      window.removeEventListener("storage", sync);
      window.removeEventListener(ROLE_CHANGE_EVENT, sync);
    };
  }, []);

  return role;
}

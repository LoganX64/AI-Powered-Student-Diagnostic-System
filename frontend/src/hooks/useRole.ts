import { useEffect, useState } from "react";
import { getTokenPayload } from "@/lib/token";

export type Role = "admin" | "coach" | "student";

export const ROLE_CHANGE_EVENT = "role-change";

function readTokenRole(): Role | null {
  const token = localStorage.getItem("admin_token");
  if (token) {
    const payload = getTokenPayload(token);
    if (payload && (payload.role === "admin" || payload.role === "coach")) {
      return payload.role;
    }
  }

  const studentToken = localStorage.getItem("student_token");
  if (studentToken) {
    const payload = getTokenPayload(studentToken);
    if (payload && payload.role === "student") {
      return "student";
    }
  }

  return null;
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

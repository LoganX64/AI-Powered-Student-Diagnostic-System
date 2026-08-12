import { useEffect, useState } from "react";
import { getTokenPayload } from "@/lib/token";

export type Role = "admin" | "coach" | "student";

export const ROLE_CHANGE_EVENT = "role-change";

function readTokenRole(): Role | null {
  const adminToken = localStorage.getItem("admin_token");
  if (adminToken) {
    const payload = getTokenPayload(adminToken);
    if (payload && payload.role === "admin") {
      return "admin";
    }
  }

  const coachToken = localStorage.getItem("coach_token");
  if (coachToken) {
    const payload = getTokenPayload(coachToken);
    if (payload && payload.role === "coach") {
      return "coach";
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

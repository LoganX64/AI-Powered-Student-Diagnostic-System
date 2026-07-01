import { useMemo } from "react";
import { getTokenPayload } from "@/lib/token";

export type Role = "admin" | "coach" | "student";

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

export function useRole(): Role {
  return useMemo(() => readTokenRole() ?? "admin", []);
}

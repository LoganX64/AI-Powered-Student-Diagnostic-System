export const STUDENT_ROUTES = ["/dashboard", "/instructions", "/quiz", "/submitted"] as const;

export const ROLE_REDIRECT_MAP = {
  admin: "/admin-signin",
  coach: "/coach-signin",
  student: "/student-login",
} as const;

export type Role = "admin" | "coach" | "student";

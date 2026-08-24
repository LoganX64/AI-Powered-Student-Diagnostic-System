export const STUDENT_ROUTES = ["/dashboard", "/instructions", "/quiz", "/submitted"] as const;

export const ROLE_REDIRECT_MAP = {
  admin: "/admin-signin",
  coach: "/coach-signin",
  student: "/student-login",
  super_admin: "/super-admin-signin",
} as const;

import { apiFetch } from "@/lib/api";

export const login = (data) =>
  apiFetch("/api/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const register = (data) =>
  apiFetch("/api/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
  });

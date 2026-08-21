import { apiFetch } from "@/lib/api";

export interface LoginPayload {
  email: string;
  password: string;
}

export interface RegisterPayload {
  email: string;
  password: string;
  org_name: string;
}

interface AuthResponse {
  token: string;
  role: string;
  tenant_id: number;
}

interface RegisterResponse {
  message: string;
  tenant_id: number;
  user_id: number;
  role: string;
}

export interface StudentLoginPayload {
  student_code: string;
}

interface StudentLoginResponse {
  token: string;
}

export const login = (data: LoginPayload) =>
  apiFetch<AuthResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const register = (data: RegisterPayload) =>
  apiFetch<RegisterResponse>("/auth/register-admin", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const loginStudent = (data: StudentLoginPayload) =>
  apiFetch<StudentLoginResponse>("/student/login", {
    method: "POST",
    body: JSON.stringify(data),
  });

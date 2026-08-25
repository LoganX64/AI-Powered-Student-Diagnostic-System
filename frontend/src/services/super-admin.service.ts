import { apiFetch } from "@/lib/api";

// --- Types ---

export type Plan = {
  id: number;
  name: string;
  slug: string;
  student_limit: number;
  coach_limit: number;
  storage_limit_bytes: number;
  test_limit: number;
  sqi_access: boolean;
  video_proctoring_included: boolean;
  video_proctoring_limit: number;
  video_proctoring_price_per_student: number;
  price_monthly: number;
  features: string[];
};

export type Tenant = {
  id: number;
  name: string;
  created_at: string;
  suspended_at: string | null;
  plan_id: number | null;
  student_count: number;
  coach_count: number;
  user_count: number;
};

export type User = {
  id: number;
  email: string;
  role: string;
  created_at: string;
};

export type GlobalStats = {
  tenants: number;
  free_tenants: number;
  paid_tenants: number;
  revenue: number;
};

export type CreateTenantPayload = {
  name: string;
  admin_email: string;
  admin_password: string;
  admin_name: string;
};

export type CreatePlanPayload = {
  name: string;
  slug: string;
  student_limit: number;
  coach_limit: number;
  storage_limit_bytes: number;
  test_limit: number;
  sqi_access: boolean;
  video_proctoring_included: boolean;
  video_proctoring_limit: number;
  video_proctoring_price_per_student: number;
  price_monthly: number;
  features: string[];
};

// --- API Functions ---

export const getGlobalStats = () =>
  apiFetch<GlobalStats>("/super-admin/stats");

export const getTenants = (params?: { search?: string; plan?: string; limit?: number; offset?: number }) => {
  const query = new URLSearchParams();
  if (params?.search) query.set("search", params.search);
  if (params?.plan) query.set("plan", params.plan);
  if (params?.limit) query.set("limit", String(params.limit));
  if (params?.offset) query.set("offset", String(params.offset));
  return apiFetch<{ total: number; data: Tenant[] }>(`/super-admin/tenants?${query}`);
};

export const createTenant = (data: CreateTenantPayload) =>
  apiFetch<{ tenant_id: number }>("/super-admin/tenants", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const getTenant = (id: number) =>
  apiFetch<Tenant>(`/super-admin/tenants/${id}`);

export const getTenantAdmins = (tenantId: number) =>
  apiFetch<{ data: User[] }>(`/super-admin/tenants/${tenantId}/admins`);

export const updateTenant = (id: number, data: { name: string }) =>
  apiFetch<{ message: string }>(`/super-admin/tenants/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const suspendTenant = (id: number) =>
  apiFetch<{ message: string }>(`/super-admin/tenants/${id}/suspend`, { method: "PUT" });

export const reactivateTenant = (id: number) =>
  apiFetch<{ message: string }>(`/super-admin/tenants/${id}/reactivate`, { method: "PUT" });

export const createTenantAdmin = (tenantId: number, data: { email: string; password: string; name: string }) =>
  apiFetch<{ user_id: number }>(`/super-admin/tenants/${tenantId}/admins`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const getPlans = () =>
  apiFetch<{ data: Plan[] }>("/super-admin/plans");

export const createPlan = (data: CreatePlanPayload) =>
  apiFetch<{ plan_id: number }>("/super-admin/plans", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const updatePlan = (id: number, data: Partial<Plan>) =>
  apiFetch<{ message: string }>(`/super-admin/plans/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const deletePlan = (id: number) =>
  apiFetch<{ message: string }>(`/super-admin/plans/${id}`, { method: "DELETE" });

export const assignPlan = (tenantId: number, planId: number) =>
  apiFetch<{ message: string }>(`/super-admin/tenants/${tenantId}/subscription`, {
    method: "PUT",
    body: JSON.stringify({ plan_id: planId }),
  });

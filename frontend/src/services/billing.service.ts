import { apiFetch } from "@/lib/api";
import type { Plan } from "@/services/super-admin.service";

export type Subscription = {
  id: number;
  tenant_id: number;
  plan_id: number;
  plan_name: string;
  status: string;
  current_period_start: string | null;
  current_period_end: string | null;
  student_count: number;
  coach_count: number;
  storage_used_bytes: number;
  overage_bytes: number;
  test_count_this_month: number;
  student_limit: number;
  coach_limit: number;
  storage_limit_bytes: number;
  test_limit: number;
  sqi_access: boolean;
  video_proctoring_included: boolean;
  video_proctoring_limit: number;
  additional_cost_paise: number;
};

export const getSubscription = () => apiFetch<Subscription>("/admin/subscription");

export const getPlans = () => apiFetch<{ data: Plan[] }>("/admin/plans");

export const createCheckout = (planSlug: string) =>
  apiFetch<{ checkout_url: string; subscription_id: string }>(
    "/admin/subscription/checkout",
    {
      method: "POST",
      body: JSON.stringify({ plan_slug: planSlug }),
    }
  );

export const cancelSubscription = () =>
  apiFetch<{ message: string }>("/admin/subscription/cancel", { method: "POST" });

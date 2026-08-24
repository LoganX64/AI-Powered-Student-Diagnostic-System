import { apiFetch } from "@/lib/api";
import { getActiveRole } from "@/lib/token";

export type Profile = {
  user_id: number;
  email: string;
  role: string;
  display_name: string | null;
  phone: string | null;
  avatar_url: string | null;
  created_at: string;
  tenant_id: number | null;
  tenant_name: string | null;
};

export type TenantSettings = {
  settings: Record<string, unknown>;
};

export type NotificationPreference = {
  id: number;
  user_id: number;
  event_type: string;
  enabled: boolean;
};

function getPrefix(): string {
  const role = getActiveRole();
  if (role === "coach") return "/coach";
  return "/admin";
}

export const getProfile = () => apiFetch<Profile>("/auth/profile");

export const updateProfile = (data: { display_name: string; phone: string }) =>
  apiFetch<{ message: string }>("/auth/profile", {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const updatePassword = (data: {
  current_password: string;
  new_password: string;
}) =>
  apiFetch<{ message: string }>("/auth/password", {
    method: "PUT",
    body: JSON.stringify(data),
  });

export const getTenantSettings = () =>
  apiFetch<TenantSettings>(`${getPrefix()}/tenant/settings`);

export const updateTenantSettings = (key: string, value: unknown) =>
  apiFetch<{ message: string }>(`${getPrefix()}/tenant/settings`, {
    method: "PUT",
    body: JSON.stringify({ key, value }),
  });

export const updateTenantName = (name: string) =>
  apiFetch<{ message: string }>(`${getPrefix()}/tenant`, {
    method: "PUT",
    body: JSON.stringify({ name }),
  });

export const getNotificationPreferences = () =>
  apiFetch<{ preferences: NotificationPreference[] }>(
    `${getPrefix()}/notifications/preferences`
  );

export const updateNotificationPreferences = (preferences: Record<string, boolean>) =>
  apiFetch<{ message: string }>(`${getPrefix()}/notifications/preferences`, {
    method: "PUT",
    body: JSON.stringify({ preferences }),
  });

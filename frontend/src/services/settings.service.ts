import { apiFetch } from "@/lib/api";
import type { Role } from "@/lib/token";
import { currentPrefix } from "@/config/routes";

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

// Route-derived, so a coach builds /coach URLs even when an admin is also signed in
// on this machine. apiFetch then picks the matching coach token from the URL.
const getPrefix = currentPrefix;

// /auth/* is shared by every staff role, so the caller names the session it means.
export const getProfile = (role: Role) =>
  apiFetch<Profile>("/auth/profile", {}, role);

export const updateProfile = (
  role: Role,
  data: { display_name: string; phone: string },
) =>
  apiFetch<{ message: string }>(
    "/auth/profile",
    {
      method: "PUT",
      body: JSON.stringify(data),
    },
    role,
  );

export const updatePassword = (
  role: Role,
  data: {
    current_password: string;
    new_password: string;
  },
) =>
  apiFetch<{ message: string }>(
    "/auth/password",
    {
      method: "PUT",
      body: JSON.stringify(data),
    },
    role,
  );

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

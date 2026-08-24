import { useState, useEffect, useCallback, useRef } from "react";
import { apiFetch } from "@/lib/api";
import { getPrefix } from "@/lib/token";

export type Notification = {
  id: number;
  tenant_id: number;
  user_id: number | null;
  event_type: string;
  title: string;
  message: string;
  priority: "info" | "warning" | "alert";
  read_at: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
};

export function useNotifications(pollInterval = 30000) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const timerRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);

  const fetchNotifications = useCallback(async () => {
    try {
      const prefix = getPrefix();
      const [notifRes, countRes] = await Promise.all([
        apiFetch<{ total: number; data: Notification[] }>(
          `${prefix}/notifications?limit=20`
        ),
        apiFetch<{ unread_count: number }>(`${prefix}/notifications/unread-count`),
      ]);
      setNotifications(notifRes.data ?? []);
      setUnreadCount(countRes.unread_count);
    } catch {
      // silently fail on poll
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchNotifications();
    timerRef.current = setInterval(fetchNotifications, pollInterval);
    return () => clearInterval(timerRef.current);
  }, [fetchNotifications, pollInterval]);

  return { notifications, unreadCount, loading, refetch: fetchNotifications };
}

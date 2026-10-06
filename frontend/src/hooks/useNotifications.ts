import { useState, useEffect, useCallback, useRef } from "react";
import { apiFetch } from "@/lib/api";
import { currentPrefix } from "@/config/routes";

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

export function useNotifications(pollInterval = 30000, enabled = true) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const timerRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const inFlightRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);

  const fetchNotifications = useCallback(async () => {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const prefix = currentPrefix();
      const [notifRes, countRes] = await Promise.all([
        apiFetch<{ total: number; data: Notification[] }>(
          `${prefix}/notifications?limit=20`,
          { signal: controller.signal },
        ),
        apiFetch<{ unread_count: number }>(`${prefix}/notifications/unread-count`, {
          signal: controller.signal,
        }),
      ]);
      setNotifications(notifRes.data ?? []);
      setUnreadCount(countRes.unread_count);
    } catch {
      // silently fail on poll
    } finally {
      inFlightRef.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!enabled) return;
    fetchNotifications();
    timerRef.current = setInterval(fetchNotifications, pollInterval);
    return () => {
      clearInterval(timerRef.current);
      abortRef.current?.abort();
    };
  }, [fetchNotifications, pollInterval, enabled]);

  return { notifications, unreadCount, loading, refetch: fetchNotifications };
}

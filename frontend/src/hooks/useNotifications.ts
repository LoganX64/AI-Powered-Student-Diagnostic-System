import { useCallback, useSyncExternalStore } from "react";
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

// Short enough that a notification feels near-immediate. This is polling rather
// than push, so it is the floor on delivery latency; the previous 30s meant a
// submitted exam could go unnoticed for half a minute.
const DEFAULT_POLL_INTERVAL = 10000;

type Snapshot = {
  notifications: Notification[];
  unreadCount: number;
  viewerUserId: number | null;
  loading: boolean;
};

const EMPTY: Snapshot = {
  notifications: [],
  unreadCount: 0,
  viewerUserId: null,
  loading: true,
};

/**
 * One poller shared by every consumer.
 *
 * The bell (in the dashboard header) and the notifications page are both mounted
 * at once on that page, so a per-component poller meant two independent intervals
 * and two sets of requests for identical data. This is a refcounted singleton:
 * the first subscriber starts the interval, the last one to leave stops it.
 *
 * `state` is mutable and holds the latest values; `snapshot` is the immutable copy
 * handed to React. React compares snapshots with Object.is and re-renders only when
 * the reference changes, so getSnapshot must return a stable reference while the
 * data is unchanged and a new one when it changes — returning the mutable object
 * directly made every poll look like "no change", leaving the list empty after a
 * reload even though the data had arrived.
 *
 * publish() therefore replaces the snapshot with a fresh copy, rather than
 * returning a new object from getSnapshot on every call, which would be a new
 * reference each time and spin React in an infinite loop.
 */
type Store = {
  state: Snapshot;
  subscribers: Set<() => void>;
  timer: ReturnType<typeof setInterval> | undefined;
  stopWake: (() => void) | undefined;
  inFlight: boolean;
  abort: AbortController | null;
  reqId: number;
  prefix: string | null;
};

let snapshot: Snapshot = EMPTY;

const store: Store = {
  state: EMPTY,
  subscribers: new Set(),
  timer: undefined,
  stopWake: undefined,
  inFlight: false,
  abort: null,
  reqId: 0,
  prefix: null,
};

function publish() {
  snapshot = { ...store.state };
  store.subscribers.forEach((cb) => cb());
}

async function fetchNow() {
  if (store.inFlight) return;
  store.inFlight = true;
  store.abort?.abort();
  const controller = new AbortController();
  store.abort = controller;
  const reqId = ++store.reqId;
  const prefix = currentPrefix();
  // A role change swaps the API prefix (see start()); recording it here keeps
  // the next fetch pointed at the same role.
  store.prefix = prefix;

  try {
    const [notifRes, countRes] = await Promise.all([
      apiFetch<{ total: number; data: Notification[]; viewer_user_id: number }>(
        `${prefix}/notifications?limit=20`,
        { signal: controller.signal },
      ),
      apiFetch<{ unread_count: number }>(`${prefix}/notifications/unread-count`, {
        signal: controller.signal,
      }),
    ]);
    // A newer request has started, so this response is stale.
    if (reqId !== store.reqId) return;
    store.state = {
      ...store.state,
      notifications: notifRes.data ?? [],
      unreadCount: countRes.unread_count,
      viewerUserId: notifRes.viewer_user_id ?? null,
      loading: false,
    };
    publish();
  } catch (err) {
    // An abort is this store replacing or tearing down its own request, not a
    // failure. Leaving state untouched either way: blanking the list on a
    // transient error is what made notifications appear to vanish.
    if ((err as Error)?.name === "AbortError") return;
  } finally {
    if (reqId === store.reqId) {
      store.inFlight = false;
    }
  }
}

function start(interval: number) {
  store.prefix = null; // force a fetch against the current prefix
  void fetchNow();
  store.timer = setInterval(() => void fetchNow(), interval);

  // Browsers throttle timers in background tabs, so refetch on return rather
  // than making the user wait out a full interval.
  const onWake = () => {
    if (document.visibilityState === "visible") void fetchNow();
  };
  window.addEventListener("focus", onWake);
  document.addEventListener("visibilitychange", onWake);
  store.stopWake = () => {
    window.removeEventListener("focus", onWake);
    document.removeEventListener("visibilitychange", onWake);
  };
}

function stop() {
  clearInterval(store.timer);
  store.timer = undefined;
  store.abort?.abort();
  // Cleared with the abort. An aborted fetch rejects asynchronously, so leaving
  // this true made the next call bail at the in-flight guard and never issue a
  // request; under StrictMode's mount -> cleanup -> mount that left the list
  // empty until the following poll.
  store.inFlight = false;
  store.stopWake?.();
  store.stopWake = undefined;
}

function subscribe(cb: () => void, interval: number) {
  store.subscribers.add(cb);
  if (store.subscribers.size === 1) start(interval);
  return () => {
    store.subscribers.delete(cb);
    if (store.subscribers.size === 0) stop();
  };
}

const getSnapshot = () => snapshot;
const getServerSnapshot = () => EMPTY;
// Stable reference: a subscribe function recreated on every render makes React
// unsubscribe and resubscribe each time, which is the documented footgun.
const noopSubscribe = () => () => {};

// The store is module state, so a hot replacement orphans the previous module's
// interval: its subscribers never unsubscribe, because the cleanup closures still
// reference the old store. Without this, editing this file during development
// leaves a poller running per save.
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    stop();
    store.subscribers.clear();
    snapshot = EMPTY;
  });
}

/**
 * Subscribes to the shared notification feed.
 *
 * `enabled=false` keeps the subscription dormant for roles with no endpoint
 * (super_admin) rather than firing requests that cannot succeed.
 */
export function useNotifications(pollInterval = DEFAULT_POLL_INTERVAL, enabled = true) {
  const subscribeCb = useCallback(
    (cb: () => void) => subscribe(cb, pollInterval),
    [pollInterval],
  );

  const current = useSyncExternalStore(
    enabled ? subscribeCb : noopSubscribe,
    enabled ? getSnapshot : getServerSnapshot,
    getServerSnapshot,
  );

  return {
    ...current,
    refetch: fetchNow,
  };
}
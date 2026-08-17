import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Counts down from `initialSeconds` and calls `onExpire` when it hits zero.
 * Persists remaining time in localStorage so a page reload or browser crash
 * doesn't reset it.
 *
 * If `examStartedAt` is provided (a timestamp in ms), the remaining time is
 * calculated from that timestamp instead of reading from storage.  This
 * enables accurate resume after browser crash / offline scenarios.
 *
 * Pass `started = false` to hold the timer on the instructions page and only
 * begin counting once the student clicks "Accept and Begin".
 *
 * Server-timing mode (tiered/premium exams): pass `serverDeadlineMs` (an
 * absolute epoch-ms deadline returned by the backend) and `serverSkewMs`
 * (clock difference between client and server). When provided, the countdown
 * is derived from the deadline on every tick instead of local storage, so it
 * is tamper-resistant and survives refreshes.
 */
export function useExamTimer(
  initialSeconds: number,
  storageKey: string,
  onExpire?: () => void,
  started: boolean = true,
  examStartedAt?: number | null,
  serverDeadlineMs?: number | null,
  serverSkewMs: number = 0,
) {
  const getInitial = useCallback(() => {
    if (serverDeadlineMs != null) {
      const now = Date.now() + serverSkewMs;
      return Math.max(0, Math.round((serverDeadlineMs - now) / 1000));
    }
    if (examStartedAt) {
      const elapsed = (Date.now() - examStartedAt) / 1000;
      return Math.max(0, Math.round(initialSeconds - elapsed));
    }
    const stored = localStorage.getItem(storageKey);
    if (stored !== null) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= 0) return parsed;
    }
    return initialSeconds;
  }, [serverDeadlineMs, serverSkewMs, examStartedAt, initialSeconds, storageKey]);

  const [timeLeft, setTimeLeft] = useState<number>(getInitial);

  // Tracks the last `initialSeconds` seen while the exam hasn't started, so the
  // timer can re-sync when the configured duration changes (e.g. data loaded
  // asynchronously on the instructions page).
  const [resyncInitial, setResyncInitial] = useState(initialSeconds);

  // Latest values are mirrored into refs so the interval effect can read them
  // without being torn down and recreated on every render or tick.
  const onExpireRef = useRef(onExpire);
  const timeLeftRef = useRef(timeLeft);
  const serverDeadlineRef = useRef(serverDeadlineMs);
  const serverSkewRef = useRef(serverSkewMs);
  const storageKeyRef = useRef(storageKey);
  useEffect(() => {
    onExpireRef.current = onExpire;
    timeLeftRef.current = timeLeft;
    serverDeadlineRef.current = serverDeadlineMs;
    serverSkewRef.current = serverSkewMs;
    storageKeyRef.current = storageKey;
  });

  // Re-sync while the exam hasn't started yet and the configured duration
  // changes (e.g. data loaded asynchronously on the instructions page). Done
  // during render (not in an effect) so the corrected value is reflected
  // immediately. Only relevant for the client-stored / local-start timer;
  // server-timing mode derives from the backend deadline on every tick.
  if (!started && serverDeadlineMs == null && initialSeconds !== resyncInitial) {
    setResyncInitial(initialSeconds);
    setTimeLeft(getInitial());
  }

  useEffect(() => {
    if (!started && serverDeadlineMs == null) return; // don't tick until exam has started

    if (timeLeftRef.current <= 0) {
      onExpireRef.current?.();
      return;
    }

    const tick = () => {
      const deadline = serverDeadlineRef.current;
      const skew = serverSkewRef.current;
      const key = storageKeyRef.current;
      if (deadline != null) {
        const now = Date.now() + skew;
        const remaining = Math.max(0, Math.round((deadline - now) / 1000));
        if (remaining <= 0) {
          onExpireRef.current?.();
          setTimeLeft(0);
        } else {
          setTimeLeft(remaining);
        }
        return;
      }
      setTimeLeft((prev) => {
        const next = prev - 1;
        localStorage.setItem(key, String(next));
        if (next <= 0) {
          onExpireRef.current?.();
        }
        return next;
      });
    };

    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [started, serverDeadlineMs]);

  return timeLeft;
}

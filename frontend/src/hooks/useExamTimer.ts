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

  // Latest values are mirrored into refs so the interval effect can read them
  // without being torn down and recreated on every render or tick.
  const onExpireRef = useRef(onExpire);
  const timeLeftRef = useRef(timeLeft);
  const serverDeadlineRef = useRef(serverDeadlineMs);
  const serverSkewRef = useRef(serverSkewMs);
  const storageKeyRef = useRef(storageKey);
  const expiredRef = useRef(false);
  useEffect(() => {
    onExpireRef.current = onExpire;
    timeLeftRef.current = timeLeft;
    serverDeadlineRef.current = serverDeadlineMs;
    serverSkewRef.current = serverSkewMs;
    storageKeyRef.current = storageKey;
  });

  useEffect(() => {
    if (!started && serverDeadlineMs == null) return; // don't tick until exam has started

    expiredRef.current = false;

    const expire = () => {
      if (expiredRef.current) return;
      expiredRef.current = true;
      onExpireRef.current?.();
    };

    if (timeLeftRef.current <= 0) {
      expire();
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
          expire();
          setTimeLeft(0);
        } else {
          setTimeLeft(remaining);
        }
        return;
      }
      const next = Math.max(0, timeLeftRef.current - 1);
      localStorage.setItem(key, String(next));
      if (next <= 0) {
        expire();
        setTimeLeft(0);
      } else {
        setTimeLeft(next);
      }
    };

    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [started, serverDeadlineMs]);

  // While the exam hasn't started (and isn't server-timed), the countdown just
  // mirrors the configured duration, which may arrive asynchronously. Deriving
  // it directly from `initialSeconds` avoids resyncing state during render or
  // in an effect. Once started (or server-timed) the ticking `timeLeft` takes over.
  const displayed =
    !started && serverDeadlineMs == null ? initialSeconds : timeLeft;
  return displayed;
}

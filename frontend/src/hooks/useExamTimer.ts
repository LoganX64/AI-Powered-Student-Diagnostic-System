import { useEffect, useRef, useState } from "react";

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
  const calcRemaining = (startedAt: number) => {
    const elapsed = (Date.now() - startedAt) / 1000;
    return Math.max(0, Math.round(initialSeconds - elapsed));
  };

  const calcFromDeadline = () => {
    if (serverDeadlineMs == null) return null;
    const now = Date.now() + serverSkewMs;
    return Math.max(0, Math.round((serverDeadlineMs - now) / 1000));
  };

  const getInitial = () => {
    if (serverDeadlineMs != null) {
      const fromDeadline = calcFromDeadline();
      if (fromDeadline != null) return fromDeadline;
    }
    // If we have a start timestamp, derive remaining from it
    if (examStartedAt) {
      return calcRemaining(examStartedAt);
    }
    // Fallback: read from localStorage (for page-refresh within same session)
    const stored = localStorage.getItem(storageKey);
    if (stored !== null) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= 0) return parsed;
    }
    return initialSeconds;
  };

  const [timeLeft, setTimeLeft] = useState<number>(getInitial);
  const onExpireRef = useRef(onExpire);
  useEffect(() => {
    onExpireRef.current = onExpire;
  });

  // Re-sync timer when initialSeconds changes before the exam starts (e.g. data
  // loaded asynchronously on the instructions page).
  useEffect(() => {
    if (!started && serverDeadlineMs == null) {
      setTimeLeft(getInitial());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialSeconds, started]);

  useEffect(() => {
    if (!started && serverDeadlineMs == null) return; // don't tick until exam has started

    if (timeLeft <= 0) {
      onExpireRef.current?.();
      return;
    }

    const id = setInterval(() => {
      if (serverDeadlineMs != null) {
        const remaining = calcFromDeadline();
        if (remaining == null || remaining <= 0) {
          onExpireRef.current?.();
          setTimeLeft(0);
        } else {
          setTimeLeft(remaining);
        }
        return;
      }
      setTimeLeft((prev) => {
        const next = prev - 1;
        localStorage.setItem(storageKey, String(next));
        if (next <= 0) {
          clearInterval(id);
          onExpireRef.current?.();
        }
        return next;
      });
    }, 1000);

    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storageKey, started, serverDeadlineMs, serverSkewMs]);

  return timeLeft;
}

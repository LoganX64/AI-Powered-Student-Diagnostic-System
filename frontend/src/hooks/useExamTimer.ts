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
 */
export function useExamTimer(
  initialSeconds: number,
  storageKey: string,
  onExpire?: () => void,
  started: boolean = true,
  examStartedAt?: number | null,
) {
  const calcRemaining = (startedAt: number) => {
    const elapsed = (Date.now() - startedAt) / 1000;
    return Math.max(0, Math.round(initialSeconds - elapsed));
  };

  const getInitial = () => {
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

  useEffect(() => {
    if (!started) return; // don't tick until exam has started

    if (timeLeft <= 0) {
      onExpireRef.current?.();
      return;
    }

    const id = setInterval(() => {
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
  }, [storageKey, started]);

  return timeLeft;
}

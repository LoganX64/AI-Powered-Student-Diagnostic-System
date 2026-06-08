import { useEffect, useRef, useState } from "react";

/**
 * Counts down from `initialSeconds` and calls `onExpire` when it hits zero.
 * Persists remaining time in sessionStorage so a page reload doesn't reset it.
 *
 * Pass `started = false` to hold the timer on the instructions page and only
 * begin counting once the student clicks "Accept and Begin".
 */
export function useExamTimer(
  initialSeconds: number,
  storageKey: string,
  onExpire?: () => void,
  started: boolean = true,
) {
  const getInitial = () => {
    const stored = sessionStorage.getItem(storageKey);
    if (stored !== null) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed) && parsed >= 0) return parsed;
    }
    return initialSeconds;
  };

  const [timeLeft, setTimeLeft] = useState<number>(getInitial);
  const onExpireRef = useRef(onExpire);
  onExpireRef.current = onExpire;

  useEffect(() => {
    if (!started) return; // don't tick until exam has started

    if (timeLeft <= 0) {
      onExpireRef.current?.();
      return;
    }

    const id = setInterval(() => {
      setTimeLeft((prev) => {
        const next = prev - 1;
        sessionStorage.setItem(storageKey, String(next));
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

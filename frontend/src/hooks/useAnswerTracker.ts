import { useCallback, useEffect, useRef, useState } from "react";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Option = "A" | "B" | "C" | "D" | "";

/** Internal record with previous_answer for change detection */
export type AnswerRecord = {
  question_id: number;
  seen: boolean;
  selected_answer: Option;
  previous_answer: Option; // internal only, stripped on submit
  time_spent: number;
  marked_for_review: boolean;
  revisited: boolean;
  changed_answer: boolean;
  was_initially_wrong: boolean;
  first_answer: Option; // snapshot of first selection (for was_initially_wrong)
};

/** Payload sent to backend (no internal fields) */
export type AnswerPayload = {
  question_id: number;
  seen: boolean;
  selected_answer: string;
  time_spent: number;
  marked_for_review: boolean;
  revisited: boolean;
  changed_answer: boolean;
  was_initially_wrong: boolean;
};

// ---------------------------------------------------------------------------
// localStorage helpers
// ---------------------------------------------------------------------------

const STORAGE_KEY = "quiz_answer_details";

function loadRecords(): Record<number, AnswerRecord> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

function saveRecords(records: Record<number, AnswerRecord>) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(records));
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useAnswerTracker(questionIds: number[]) {
  const [records, setRecords] = useState<Record<number, AnswerRecord>>(
    loadRecords,
  );

  // Timer refs — mutable, no re-renders
  const activeQuestionIdRef = useRef<number | null>(null);
  const tickStartRef = useRef<number>(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Persist to localStorage on every state change
  useEffect(() => {
    saveRecords(records);
  }, [records]);

  // Cleanup interval on unmount
  useEffect(() => {
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  // ---------------------------------------------------------------------------
  // Actions
  // ---------------------------------------------------------------------------

  /** Mark question as seen. If already seen before, mark as revisited. */
  const markSeen = useCallback(
    (id: number) => {
      setRecords((prev) => {
        const existing = prev[id];
        const record: AnswerRecord = existing
          ? {
              ...existing,
              seen: true,
              revisited: existing.seen, // revisited if was already seen
            }
          : {
              question_id: id,
              seen: true,
              selected_answer: "",
              previous_answer: "",
              time_spent: 0,
              marked_for_review: false,
              revisited: false,
              changed_answer: false,
              was_initially_wrong: false,
              first_answer: "",
            };
        return { ...prev, [id]: record };
      });
    },
    [],
  );

  /** Select an answer for a question. Detects answer changes. */
  const selectAnswer = useCallback(
    (id: number, option: Option) => {
      setRecords((prev) => {
        const existing = prev[id];
        const isFirstSelection =
          existing && existing.selected_answer === "" && option !== "";

        const record: AnswerRecord = {
          ...(existing || {
            question_id: id,
            seen: true,
            selected_answer: "",
            previous_answer: "",
            time_spent: 0,
            marked_for_review: false,
            revisited: false,
            changed_answer: false,
            was_initially_wrong: false,
            first_answer: "",
          }),
          seen: true,
          previous_answer: existing?.selected_answer || "",
          selected_answer: option,
          // Track first answer snapshot
          first_answer:
            isFirstSelection || !existing
              ? option
              : existing.first_answer || option,
          // Detect answer change
          changed_answer:
            existing && existing.selected_answer !== "" && existing.selected_answer !== option
              ? true
              : existing?.changed_answer || false,
        };

        return { ...prev, [id]: record };
      });
    },
    [],
  );

  /** Toggle mark-for-review flag. */
  const toggleMarkForReview = useCallback(
    (id: number) => {
      setRecords((prev) => {
        const existing = prev[id];
        if (!existing) return prev;
        return {
          ...prev,
          [id]: {
            ...existing,
            marked_for_review: !existing.marked_for_review,
          },
        };
      });
    },
    [],
  );

  /** Start tracking time for a question (call when navigating to it). */
  const startTracking = useCallback(
    (id: number) => {
      // Stop any existing interval
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }

      activeQuestionIdRef.current = id;
      tickStartRef.current = Date.now();

      // Tick every second to accumulate time
      intervalRef.current = setInterval(() => {
        const elapsed = (Date.now() - tickStartRef.current) / 1000;
        tickStartRef.current = Date.now();

        setRecords((prev) => {
          const current = activeQuestionIdRef.current;
          if (current === null || !prev[current]) return prev;
          return {
            ...prev,
            [current]: {
              ...prev[current],
              time_spent: prev[current].time_spent + elapsed,
            },
          };
        });
      }, 1000);
    },
    [],
  );

  /** Stop tracking time (call when navigating away). */
  const stopTracking = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    activeQuestionIdRef.current = null;
  }, []);

  /** Get the final payload array for backend submission. */
  const getPayload = useCallback(
    (ids: number[]): AnswerPayload[] => {
      return ids.map((id) => {
        const r = records[id];
        if (!r) {
          // Question was never seen
          return {
            question_id: id,
            seen: false,
            selected_answer: "",
            time_spent: 0,
            marked_for_review: false,
            revisited: false,
            changed_answer: false,
            was_initially_wrong: false,
          };
        }
        return {
          question_id: r.question_id,
          seen: r.seen,
          selected_answer: r.selected_answer,
          time_spent: Math.round(r.time_spent * 100) / 100, // round to 2 decimals
          marked_for_review: r.marked_for_review,
          revisited: r.revisited,
          changed_answer: r.changed_answer,
          was_initially_wrong: r.was_initially_wrong,
        };
      });
    },
    [records],
  );

  /** Clear all tracking data (on submit). */
  const clearAll = useCallback(() => {
    stopTracking();
    setRecords({});
    localStorage.removeItem(STORAGE_KEY);
  }, [stopTracking]);

  // ---------------------------------------------------------------------------
  // Derived state for convenience
  // ---------------------------------------------------------------------------

  const answeredCount = questionIds.filter(
    (id) => records[id]?.selected_answer !== "",
  ).length;

  const markedForReviewIds = questionIds.filter(
    (id) => records[id]?.marked_for_review,
  );

  const seenIds = questionIds.filter((id) => records[id]?.seen);

  return {
    records,
    markSeen,
    selectAnswer,
    toggleMarkForReview,
    startTracking,
    stopTracking,
    getPayload,
    clearAll,
    answeredCount,
    markedForReviewIds,
    seenIds,
  };
}

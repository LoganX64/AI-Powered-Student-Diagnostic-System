import { useCallback, useEffect, useRef, useState } from "react";
import type { AutosaveAnswer, AnswerPayload } from "@/services/student.service";

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
  first_answer: Option; // snapshot of first selection (for backend to derive was_initially_wrong)
};

// ---------------------------------------------------------------------------
// localStorage helpers
// ---------------------------------------------------------------------------

const STORAGE_KEY_PREFIX = "quiz_answer_details_";

function loadRecords(storageKey: string): Record<number, AnswerRecord> {
  try {
    const raw = localStorage.getItem(storageKey);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

function saveRecords(storageKey: string, records: Record<number, AnswerRecord>) {
  try {
    localStorage.setItem(storageKey, JSON.stringify(records));
  } catch {
    // QuotaExceededError (or storage disabled) must not kill the tracker.
  }
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useAnswerTracker(questionIds: number[], assignmentId: number) {
  // Answers are namespaced per assignment so exams on the same device can never
  // read each other's records.
  const storageKey = `${STORAGE_KEY_PREFIX}${assignmentId}`;

  const [records, setRecords] = useState<Record<number, AnswerRecord>>(() =>
    loadRecords(storageKey),
  );

  // Timer refs — mutable, no re-renders
  const activeQuestionIdRef = useRef<number | null>(null);
  const segmentStartRef = useRef(0);
  const committedTimeRef = useRef(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const recordsRef = useRef(records);

  // Persist to localStorage on every state change
  useEffect(() => {
    saveRecords(storageKey, records);
  }, [records, storageKey]);

  // Keep a ref mirror of records so the []-deps callbacks can read latest values
  useEffect(() => {
    recordsRef.current = records;
  }, [records]);

  // Flush the active question's running time segment on unmount
  useEffect(() => {
    return () => {
      const id = activeQuestionIdRef.current;
      if (id !== null && intervalRef.current !== null) {
        const elapsed = (Date.now() - segmentStartRef.current) / 1000;
        const total = committedTimeRef.current + elapsed;
        setRecords((prev) => {
          const r = prev[id];
          if (!r) return prev;
          return { ...prev, [id]: { ...r, time_spent: total } };
        });
      }
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

  /** Commit the active question's running time segment into its record. */
  const commitActiveSegment = useCallback(() => {
    const id = activeQuestionIdRef.current;
    if (id === null || intervalRef.current === null) return;
    const elapsed = (Date.now() - segmentStartRef.current) / 1000;
    const total = committedTimeRef.current + elapsed;
    setRecords((prev) => {
      const r = prev[id];
      if (!r) return prev;
      return { ...prev, [id]: { ...r, time_spent: total } };
    });
    clearInterval(intervalRef.current);
    intervalRef.current = null;
    activeQuestionIdRef.current = null;
  }, []);

  /** Start tracking time for a question (call when navigating to it). */
  const startTracking = useCallback(
    (id: number) => {
      // Clear any stale interval, then commit the previous question's segment
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      if (activeQuestionIdRef.current !== null) {
        commitActiveSegment();
      }

      activeQuestionIdRef.current = id;
      committedTimeRef.current = recordsRef.current[id]?.time_spent ?? 0;
      segmentStartRef.current = Date.now();

      // Tick to persist/refresh. Time is measured from the fixed anchor, so
      // accuracy is independent of tick cadence (no drift, no lost seconds).
      intervalRef.current = setInterval(() => {
        const current = activeQuestionIdRef.current;
        if (current === null) return;
        const elapsed = (Date.now() - segmentStartRef.current) / 1000;
        setRecords((prev) => {
          const r = prev[current];
          if (!r) return prev;
          return {
            ...prev,
            [current]: {
              ...r,
              time_spent: committedTimeRef.current + elapsed,
            },
          };
        });
      }, 1000);
    },
    [commitActiveSegment],
  );

  /** Stop tracking time (call when navigating away). */
  const stopTracking = useCallback(() => {
    commitActiveSegment();
  }, [commitActiveSegment]);

  /** Get the final payload array for backend submission. */
  const getPayload = useCallback(
    (ids: number[]): AnswerPayload[] => {
      // Read through recordsRef so this callback's identity never changes. `records`
      // changes every second (time-tracking tick), which would otherwise give this
      // callback a new identity each second and cascade re-creations up into the
      // autosave interval.
      const recs = recordsRef.current;
      return ids.map((id) => {
        const r = recs[id];
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
            first_answer: "",
          };
        }
        return {
          question_id: r.question_id,
          seen: r.seen,
          selected_answer: r.selected_answer,
          time_spent: r.time_spent, // seconds (must match autosave path)
          marked_for_review: r.marked_for_review,
          revisited: r.revisited,
          changed_answer: r.changed_answer,
          first_answer: r.first_answer ?? "",
        };
      });
    },
    [],
  );

  /** Overwrite the current records (used to resume a saved attempt). */
  const restoreRecords = useCallback(
    (next: Record<number, AnswerRecord>) => {
      setRecords(next);
    },
    [],
  );

  /** Build the payload used by the server autosave endpoint. */
  const getAutosavePayload = useCallback((): AutosaveAnswer[] => {
    // recordsRef (not `records`) so this callback is stable for the component's
    // lifetime. Otherwise the interval would post a payload frozen at the render
    // that created it, and would never send the student's later answers.
    return Object.values(recordsRef.current).map((r) => ({
      question_id: r.question_id,
      selected_answer: r.selected_answer,
      seen: r.seen,
      time_spent: r.time_spent, // seconds
      marked_for_review: r.marked_for_review,
      revisited: r.revisited,
      changed_answer: r.changed_answer,
      first_answer: r.first_answer ?? "",
    }));
  }, []);

  // ---------------------------------------------------------------------------
  // Derived state for convenience
  // ---------------------------------------------------------------------------

  // `?? ""` is load-bearing: a never-visited question has no record at all, and
  // `undefined !== ""` would count it as answered.
  const answeredCount = questionIds.filter(
    (id) => (records[id]?.selected_answer ?? "") !== "",
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
    restoreRecords,
    getAutosavePayload,
    answeredCount,
    markedForReviewIds,
    seenIds,
  };
}

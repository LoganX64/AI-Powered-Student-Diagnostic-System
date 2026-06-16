import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ExamHeader } from "../../components/student/exam-header";
import { Button } from "../../components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "../../components/ui/alert-dialog";
import { cn } from "../../lib/utils";
import { useExamTimer } from "../../hooks/useExamTimer";
import { useAnswerTracker } from "../../hooks/useAnswerTracker";
import { submitAnswers } from "../../services/student.service";
import { Flag } from "lucide-react";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Option = "A" | "B" | "C" | "D";

type Question = {
  id: number;
  text: string;
  imageUrl?: string;
  options: Record<Option, string>;
};

// ---------------------------------------------------------------------------
// Sample data  (replace with real API fetch)
// ---------------------------------------------------------------------------

const SAMPLE_QUESTIONS: Question[] = [
  {
    id: 1,
    text: "Which of the following is the capital of France?",
    options: { A: "Berlin", B: "Madrid", C: "Paris", D: "Rome" },
  },
  {
    id: 2,
    text: "Which number is even?",
    options: { A: "7", B: "11", C: "12", D: "9" },
  },
  {
    id: 3,
    text: "What is 5 + 3?",
    options: { A: "7", B: "8", C: "10", D: "9" },
  },
  {
    id: 4,
    text: "Which planet is closest to the Sun?",
    options: { A: "Venus", B: "Earth", C: "Mars", D: "Mercury" },
  },
  {
    id: 5,
    text: "What is the square root of 144?",
    options: { A: "10", B: "11", C: "12", D: "13" },
  },
  {
    id: 6,
    text: "H₂O is the chemical formula for?",
    options: { A: "Oxygen", B: "Hydrogen", C: "Water", D: "Hydrogen peroxide" },
  },
];

const EXAM_DURATION_SECONDS = 60 * 60;

// Hardcoded assignment ID (replace with real assignment flow later)
const MOCK_ASSIGNMENT_ID = 1;

// ---------------------------------------------------------------------------
// Helpers — localStorage persistence (for current index only)
// ---------------------------------------------------------------------------

function loadCurrentIndex(): number {
  const raw = localStorage.getItem("current_question_index");
  if (raw !== null) {
    const parsed = parseInt(raw, 10);
    if (!isNaN(parsed) && parsed >= 0) return parsed;
  }
  return 0;
}

function saveCurrentIndex(index: number) {
  localStorage.setItem("current_question_index", String(index));
}

function clearExamStorage() {
  localStorage.removeItem("quiz_answers");
  localStorage.removeItem("quiz_answer_details");
  localStorage.removeItem("current_question_index");
  localStorage.removeItem("exam_started");
  localStorage.removeItem("exam_started_at");
  localStorage.removeItem("exam_timer");
  localStorage.removeItem("pending_submission");
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function StudentQuizPage() {
  const navigate = useNavigate();

  const studentCode = useMemo(
    () => localStorage.getItem("student_code") || "",
    [],
  );

  const [questions] = useState<Question[]>(SAMPLE_QUESTIONS);
  const [currentIndex, setCurrentIndex] = useState<number>(loadCurrentIndex);
  const [submitting, setSubmitting] = useState(false);
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const isAutoSubmitRef = useRef(false);

  const questionIds = useMemo(() => questions.map((q) => q.id), [questions]);

  const {
    records,
    markSeen,
    selectAnswer,
    toggleMarkForReview,
    startTracking,
    stopTracking,
    getPayload,
    answeredCount,
    markedForReviewIds,
  } = useAnswerTracker(questionIds);

  const currentQuestion = questions[currentIndex];
  const currentRecord = records[currentQuestion.id];

  // Track question changes — mark seen + start/stop timer
  useEffect(() => {
    const qId = currentQuestion.id;
    markSeen(qId);
    startTracking(qId);

    return () => {
      stopTracking();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentIndex]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      stopTracking();
    };
  }, [stopTracking]);

  // ---------------------------------------------------------------------------
  // Submit handler
  // ---------------------------------------------------------------------------

  const performSubmit = useCallback(async () => {
    if (submitting) return;
    setSubmitting(true);
    stopTracking();

    const payload = getPayload(questionIds);

    try {
      await submitAnswers(MOCK_ASSIGNMENT_ID, payload);
      clearExamStorage();
      navigate("/submitted", { replace: true });
    } catch {
      // Queue for retry — store in localStorage
      localStorage.setItem(
        "pending_submission",
        JSON.stringify({
          assignment_id: MOCK_ASSIGNMENT_ID,
          answers: payload,
          queued_at: Date.now(),
        }),
      );
      clearExamStorage();
      navigate("/submitted", { replace: true });
    }
  }, [submitting, stopTracking, getPayload, questionIds, navigate]);

  /** Called when user manually clicks Submit button — shows confirm dialog if time remains */
  const handleManualSubmit = () => {
    isAutoSubmitRef.current = false;
    setShowConfirmDialog(true);
  };

  /** Called by timer on expiry — submits directly without dialog */
  const handleAutoSubmit = () => {
    isAutoSubmitRef.current = true;
    performSubmit();
  };

  // ---------------------------------------------------------------------------
  // Timer
  // ---------------------------------------------------------------------------

  const examStarted = localStorage.getItem("exam_started") === "true";
  const examStartedAtRaw = localStorage.getItem("exam_started_at");
  const examStartedAt = examStartedAtRaw ? Number(examStartedAtRaw) : null;

  const timerTimeLeft = useExamTimer(
    EXAM_DURATION_SECONDS,
    "exam_timer",
    () => {
      handleAutoSubmit();
    },
    examStarted,
    examStartedAt,
  );

  const timeLeft = timerTimeLeft;

  // ---------------------------------------------------------------------------
  // Navigation handlers
  // ---------------------------------------------------------------------------

  const handleSelect = (option: Option) => {
    selectAnswer(currentQuestion.id, option);
  };

  const handleNext = () => {
    if (currentIndex < questions.length - 1) {
      const next = currentIndex + 1;
      setCurrentIndex(next);
      saveCurrentIndex(next);
    }
  };

  const handleNavigate = (index: number) => {
    setCurrentIndex(index);
    saveCurrentIndex(index);
  };

  const handleMarkForReview = () => {
    toggleMarkForReview(currentQuestion.id);
  };

  // ---------------------------------------------------------------------------
  // Derived state
  // ---------------------------------------------------------------------------

  const isLast = currentIndex === questions.length - 1;
  const isMarked = currentRecord?.marked_for_review ?? false;

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="flex min-h-screen flex-col bg-background px-4 py-6 sm:px-8">
      {/* Header */}
      <ExamHeader candidateName={studentCode} timeLeft={timeLeft} />

      {/* Body */}
      <div className="mt-6 flex flex-1 gap-4">
        {/* ---- Left panel: question ---- */}
        <div className="flex flex-1 flex-col rounded-2xl border border-border bg-card shadow-sm">
          {/* Question text */}
          <div className="border-b border-border px-7 py-5">
            <p className="text-xs font-medium text-muted-foreground">
              Question {currentIndex + 1} of {questions.length}
            </p>
            <p className="mt-2 text-sm font-medium leading-relaxed text-foreground">
              {currentQuestion.text}
            </p>
          </div>

          {/* Diagram area */}
          {currentQuestion.imageUrl && (
            <div className="border-b border-border px-7 py-4">
              <div className="flex min-h-45 items-center justify-center rounded-xl border border-dashed border-border bg-muted/40">
                <img
                  src={currentQuestion.imageUrl}
                  alt={`Diagram for question ${currentIndex + 1}`}
                  className="max-h-70 max-w-full rounded-lg object-contain"
                />
              </div>
            </div>
          )}

          {/* Options */}
          <div className="flex-1 px-7 py-5">
            <div className="space-y-3">
              {(["A", "B", "C", "D"] as Option[]).map((opt) => {
                const isSelected = currentRecord?.selected_answer === opt;
                return (
                  <label
                    key={opt}
                    className={cn(
                      "flex cursor-pointer items-center gap-3 rounded-xl border px-4 py-3 text-sm transition-colors",
                      isSelected
                        ? "border-primary bg-primary/5 text-primary"
                        : "border-border bg-card text-foreground hover:bg-muted/60",
                    )}
                  >
                    <input
                      type="radio"
                      name={`question-${currentQuestion.id}`}
                      value={opt}
                      checked={isSelected}
                      onChange={() => handleSelect(opt)}
                      className="accent-primary"
                      aria-label={`Option ${opt}`}
                    />
                    <span className="font-semibold text-muted-foreground w-5">
                      {opt}.
                    </span>
                    <span>{currentQuestion.options[opt]}</span>
                  </label>
                );
              })}
            </div>
          </div>

          {/* Mark for Review + Navigation */}
          <div className="border-t border-border px-7 py-4 flex items-center justify-between">
            <Button
              onClick={handleMarkForReview}
              variant={isMarked ? "default" : "outline"}
              className={cn(
                "gap-2",
                isMarked &&
                  "bg-yellow-500 hover:bg-yellow-600 text-white border-yellow-500",
              )}
            >
              <Flag className="h-4 w-4" />
              {isMarked ? "Unmark Review" : "Mark for Review"}
            </Button>

            {isLast ? (
              <Button
                onClick={handleManualSubmit}
                disabled={submitting}
                variant="default"
              >
                {submitting ? "Submitting..." : "Submit"}
              </Button>
            ) : (
              <Button onClick={handleNext} variant="default">
                Next
              </Button>
            )}
          </div>
        </div>

        {/* ---- Right sidebar ---- */}
        <div className="flex w-48 shrink-0 flex-col rounded-2xl border border-border bg-card shadow-sm sm:w-52">
          {/* Question navigator grid */}
          <div className="flex-1 p-4">
            <p className="mb-3 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Questions
            </p>
            <div className="grid grid-cols-3 gap-2">
              {questions.map((q, index) => {
                const record = records[q.id];
                const answered = (record?.selected_answer ?? "") !== "";
                const marked = record?.marked_for_review ?? false;
                const isCurrent = index === currentIndex;

                return (
                  <button
                    key={q.id}
                    onClick={() => handleNavigate(index)}
                    aria-label={`Go to question ${index + 1}`}
                    className={cn(
                      "flex h-9 w-9 items-center justify-center rounded-lg text-xs font-semibold transition-colors",
                      isCurrent
                        ? "bg-primary text-primary-foreground"
                        : marked
                          ? "bg-yellow-100 text-yellow-700 border border-yellow-300 dark:bg-yellow-900/30 dark:text-yellow-400 dark:border-yellow-700"
                          : answered
                            ? "bg-green-100 text-green-700 border border-green-300 dark:bg-green-900/30 dark:text-green-400 dark:border-green-700"
                            : "bg-muted text-muted-foreground hover:bg-muted/80 border border-border",
                    )}
                  >
                    {index + 1}
                  </button>
                );
              })}
            </div>

            {/* Legend */}
            <div className="mt-4 space-y-1.5">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="h-3 w-3 rounded bg-primary" />
                Current
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="h-3 w-3 rounded bg-yellow-200 border border-yellow-300" />
                Marked for Review
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="h-3 w-3 rounded bg-green-200 border border-green-300" />
                Answered
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="h-3 w-3 rounded bg-muted border border-border" />
                Not answered
              </div>
            </div>

            {/* Progress */}
            <p className="mt-4 text-xs text-muted-foreground">
              {answeredCount}/{questions.length} answered
            </p>
            {markedForReviewIds.length > 0 && (
              <p className="mt-1 text-xs text-yellow-600">
                {markedForReviewIds.length} marked for review
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Submit confirmation dialog */}
      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Submit Exam?</AlertDialogTitle>
            <AlertDialogDescription className="space-y-2">
              <p>
                You have{" "}
                <span className="font-semibold text-foreground">
                  {Math.floor(timeLeft / 60)}m {timeLeft % 60}s
                </span>{" "}
                remaining. Are you sure you want to submit your exam?
              </p>
              <p>Once submitted, you will not be able to:</p>
              <ul className="list-disc pl-5 text-muted-foreground">
                <li>Change any answers</li>
                <li>Review marked questions</li>
                <li>Return to the exam</li>
              </ul>
              <p className="font-medium text-foreground">
                {answeredCount} of {questions.length} questions answered.
              </p>
              {markedForReviewIds.length > 0 && (
                <p className="text-yellow-600">
                  {markedForReviewIds.length} question(s) marked for review.
                </p>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Continue Exam</AlertDialogCancel>
            <AlertDialogAction
              onClick={performSubmit}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Yes, Submit Exam
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ExamHeader } from "../../components/student/exam-header";
import { Button } from "../../components/ui/button";
import { cn } from "../../lib/utils";
import { useExamTimer } from "../../hooks/useExamTimer";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Option = "A" | "B" | "C" | "D";

type Question = {
  id: number;
  text: string;
  /** Optional URL to a diagram/image for the question */
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

// Exam duration - reuse same key so the timer is continuous from instructions
const EXAM_DURATION_SECONDS = 60 * 60;

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function StudentQuizPage() {
  const navigate = useNavigate();

  const studentCode = useMemo(
    () => localStorage.getItem("student_code") || "",
    [],
  );

  // Guard: redirect to login if no token
  useEffect(() => {
    const token = localStorage.getItem("student_token");
    if (!token) {
      navigate("/", { replace: true });
    }
  }, [navigate]);

  const [questions] = useState<Question[]>(SAMPLE_QUESTIONS);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<Record<number, Option | null>>({});

  const currentQuestion = questions[currentIndex];

  // Timer — continuous from instruction page (same storage key)
  const timeLeft = useExamTimer(EXAM_DURATION_SECONDS, "exam_timer", () => {
    handleSubmit();
  });

  const handleSelect = (option: Option) => {
    setAnswers((prev) => ({ ...prev, [currentQuestion.id]: option }));
  };

  const handleNext = () => {
    if (currentIndex < questions.length - 1) {
      setCurrentIndex((i) => i + 1);
    }
  };

  const handleNavigate = (index: number) => {
    setCurrentIndex(index);
  };

  const handleSubmit = () => {
    // Store answers so the submitted page (or future API call) can access them
    sessionStorage.setItem("quiz_answers", JSON.stringify(answers));
    // Clear exam timer storage
    sessionStorage.removeItem("exam_timer");
    navigate("/submitted", { replace: true });
  };

  const isLast = currentIndex === questions.length - 1;
  const answeredCount = Object.values(answers).filter(Boolean).length;

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

          {/* Diagram area (shown only when imageUrl exists) */}
          {currentQuestion.imageUrl && (
            <div className="border-b border-border px-7 py-4">
              <div className="flex min-h-[180px] items-center justify-center rounded-xl border border-dashed border-border bg-muted/40">
                <img
                  src={currentQuestion.imageUrl}
                  alt={`Diagram for question ${currentIndex + 1}`}
                  className="max-h-[280px] max-w-full rounded-lg object-contain"
                />
              </div>
            </div>
          )}

          {/* Options */}
          <div className="flex-1 px-7 py-5">
            <div className="space-y-3">
              {(["A", "B", "C", "D"] as Option[]).map((opt) => {
                const isSelected = answers[currentQuestion.id] === opt;
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
                const answered = !!answers[q.id];
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
          </div>

          {/* Next / Submit button */}
          <div className="border-t border-border p-4">
            {isLast ? (
              <Button
                onClick={handleSubmit}
                className="w-full"
                variant="default"
              >
                Submit
              </Button>
            ) : (
              <Button onClick={handleNext} className="w-full" variant="default">
                Next
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
